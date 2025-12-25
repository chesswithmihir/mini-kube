# Mini-Kube Project

This project aims to demystify Docker and Kubernetes by building them from scratch. It provides a hands-on approach to understanding the fundamental Linux kernel concepts that power containerization and orchestration. By building simplified versions of `runc` (container runtime) and Kubernetes (orchestrator), we explore the low-level mechanics that often remain hidden behind high-level tools.

## Part 1: `my-runc` - The Container Runtime

`my-runc` is our simplified implementation of `runc`, the OCI (Open Container Initiative) compliant runtime that Docker and other container platforms use to spawn and manage containers. It focuses on demonstrating three core Linux kernel features essential for container isolation and resource management: Namespaces, Control Groups (cgroups), and `pivot_root`.

### Core Concepts Explained: The Building Blocks of Containers

To understand `my-runc`, we first need to grasp the foundational Linux kernel features it leverages. We will explain these concepts assuming zero prior knowledge of operating systems, diving deep into their mechanisms and how they contribute to containerization.

#### 1. Linux Kernel Namespaces: The Isolation Mechanism

**What are Namespaces?**

In Linux, namespaces are a powerful feature that partitions global system resources such that processes within a namespace see an isolated instance of that resource. Imagine a large apartment building (the host system). Each apartment (a container) has its own set of amenities (process IDs, network interfaces, file system mounts, etc.), even though they all exist within the same physical building. Namespaces provide this "apartment" view to processes, preventing them from seeing or interfering with the resources of other "apartments" or the building management (the host system).

**How do Namespaces Work?**

When a new process is created, it inherits the namespaces of its parent. To create a container, we essentially tell the kernel to create new, isolated namespaces for the child process. This is achieved primarily through two Linux system calls: `unshare()` and `clone()`.

*   **`unshare(CLONE_NEW* flags)`:** This syscall allows a process to disassociate parts of its execution context (like namespaces) from its parent. For example, `unshare(CLONE_NEWPID)` tells the kernel to move the calling process into a new PID namespace.
*   **`clone(CLONE_NEW* flags, ...)`:** This syscall creates a new child process, similar to `fork()`, but with more control. By passing `CLONE_NEW*` flags, we can specify which new namespaces the child process should be created within. Our `my-runc` uses `clone()` implicitly via `os/exec.Cmd.SysProcAttr.Cloneflags` to create the child process in new namespaces.

Let's explore the specific namespace types implemented in `my-runc`:

*   **PID Namespace (`CLONE_NEWPID`): Process ID Isolation**
    *   **What it does:** Isolates the process ID (PID) space. Inside a PID namespace, processes are assigned PIDs starting from 1, independent of the host's PID numbering. The first process in a new PID namespace gets PID 1, just like `init` or `systemd` on the host. This process becomes the "init" for that namespace and handles orphaned processes.
    *   **Why it's crucial for containers:** Without PID isolation, a process inside a container might see and potentially kill host processes, or its PIDs would conflict with host PIDs, making process management impossible. PID 1 inside the container is critical for proper process supervision.

*   **Mount Namespace (`CLONE_NEWNS`): Isolated Filesystem Views**
    *   **What it does:** Isolates the list of mount points seen by processes. Each mount namespace has its own view of the filesystem hierarchy. Changes to mount points (e.g., mounting a new filesystem, unmounting) within one mount namespace are not visible in others.
    *   **Why it's crucial for containers:** This is fundamental for providing a container with its own isolated root filesystem, preventing it from accessing or modifying the host's filesystem directly. This is often combined with `pivot_root`.

*   **UTS Namespace (`CLONE_NEWUTS`): Isolated Hostname and Domainname**
    *   **What it does:** Isolates the hostname and NIS (Network Information Service) domain name. Processes in different UTS namespaces can have different hostnames.
    *   **Why it's crucial for containers:** Allows each container to have its own identity on the network (e.g., its own hostname) without affecting the host or other containers.

*   **IPC Namespace (`CLONE_NEWIPC`): Isolated Inter-Process Communication**
    *   **What it does:** Isolates inter-process communication (IPC) resources such as System V IPC objects (message queues, semaphores, shared memory segments) and POSIX message queues.
    *   **Why it's crucial for containers:** Prevents processes in one container from interfering with IPC mechanisms used by other containers or the host, ensuring communication isolation.

*   **Network Namespace (`CLONE_NEWNET`): Isolated Network Stack**
    *   **What it does:** Provides an isolated network stack. This includes its own network devices (e.g., `lo`, `eth0`), IP addresses, routing tables, firewall rules, and port numbers.
    *   **Why it's crucial for containers:** Each container can have its own virtual network interface, IP address, and port mappings, making it appear as a separate machine on the network. This allows multiple containers to run on the same host and bind to the same port numbers (e.g., port 80) without conflict.

*   **User Namespace (`CLONE_NEWUSER`): Isolated User and Group IDs**
    *   **What it does:** Isolates the user and group ID (UID/GID) space. This allows a process to have root privileges (UID 0) inside the container while being mapped to an unprivileged UID on the host.
    *   **Why it's crucial for containers:** A critical security feature. It enables processes within a container to run as `root` for administrative tasks without actually having `root` privileges on the host system, significantly reducing the security blast radius if a container is compromised. Our `my-runc` uses UID/GID mapping to achieve this.

#### 2. Linux Control Groups (cgroups): Resource Management

**What are cgroups?**

Control Groups (cgroups) are a Linux kernel feature that allows for the allocation, prioritization, and management of system resources (CPU, memory, disk I/O, network) among groups of processes. While namespaces isolate *what* a process can see, cgroups control *how much* resources a process (or a group of processes) can use.

**How do cgroups Work? (Architecture)**

Cgroups are organized hierarchically, similar to a filesystem. The kernel exposes a virtual filesystem (typically mounted at `/sys/fs/cgroup`) where directories represent cgroups and special files within those directories allow administrators to configure resource limits.

*   **Hierarchy:** Cgroups form a tree structure. A child cgroup can further subdivide the resources allocated to its parent.
*   **Subsystems:** Different resource types (CPU, memory, blkio, net_cls, etc.) are managed by different "subsystems."

**Versioning: Cgroups v1 vs. Cgroups v2**

Modern Linux kernels have moved to **Cgroups v2**, which offers a unified hierarchy for all resource controllers, unlike v1 which had separate hierarchies for memory, cpu, etc. `my-runc` is robust enough to detect and handle both versions.

*   **Cgroups v1 Strategy:**
    *   Checks for directory `/sys/fs/cgroup/memory`.
    *   Creates a new cgroup directory: `/sys/fs/cgroup/memory/my-container`.
    *   **Enabling:** Writes the process PID to `cgroup.procs`.
    *   **Limiting:** Writes the limit in bytes to `memory.limit_in_bytes`.

*   **Cgroups v2 Strategy:**
    *   Checks for the unified hierarchy at `/sys/fs/cgroup` (where `memory.max` exists).
    *   Creates a new cgroup directory: `/sys/fs/cgroup/my-container`.
    *   **Enabling:** Writes the process PID to `cgroup.procs`.
    *   **Limiting:** Writes the limit in bytes to `memory.max`.

**Implementation Details (`setupCgroups`):**
Our implementation in `namespace_linux.go` dynamically detects the cgroup version by checking for the existence of the `memory` controller directory. This ensures `my-runc` works on both older systems and modern distributions like Ubuntu 22.04+ (used in Lima VMs).

#### 3. Root Filesystem Isolation (`pivot_root`): Changing the Container's View of the World

**What is `pivot_root`?**

`pivot_root` is a Linux system call that changes the root filesystem of the current process and all its children. Unlike `chroot()`, which only changes the root directory for a process and its children but doesn't fully detach the process from the old root filesystem, `pivot_root()` completely moves the current process's root from the old root to a new directory. The old root is then typically unmounted and becomes inaccessible to the process.

**The `pivot_root` Dance (Detailed Implementation):**

Performing `pivot_root` correctly requires a specific sequence of operations to satisfy kernel requirements (specifically, that the new root must be a mount point separate from the old root).

1.  **Prepare the New Root Location:** We create a temporary directory `/tmp/my-runc-root` to serve as the staging area for our new root.
2.  **Bind Mount the Root Filesystem:** We perform a bind mount of the desired root filesystem (in our simple case, the host's `/`) onto `/tmp/my-runc-root`. This effectively clones the filesystem view into that directory.
    *   `syscall.Mount(rootfs, newRoot, "", syscall.MS_BIND|syscall.MS_REC, "")`
3.  **Make it Private:** We mark this new mount as "private". This ensures that mount events within this new namespace don't propagate back to the host namespace.
    *   `syscall.Mount(newRoot, newRoot, "", syscall.MS_PRIVATE|syscall.MS_REC, "")`
4.  **Prepare the "Old Root" Holding Area:** We create a directory inside our new root, e.g., `/tmp/my-runc-root/.pivot_root`, to temporarily hold the old filesystem.
5.  **Execute `pivot_root`:** We call the syscall.
    *   `syscall.PivotRoot(newRoot, putOld)`
    *   At this exact moment, `/tmp/my-runc-root` becomes `/`. The old `/` is moved to `/.pivot_root`.
6.  **Switch Working Directory:** The process is technically still "standing" in the old directory structure. We explicitly `os.Chdir("/")` to move into the top of the new root.
7.  **Unmount the Old Root:** We unmount `/.pivot_root` to sever the link to the host completely.
    *   `syscall.Unmount("/.pivot_root", syscall.MNT_DETACH)`
8.  **Mount `/proc`:** This is a critical final step. Process tools like `ps` rely on the `/proc` filesystem to list running processes. If we don't mount a fresh version of `/proc` inside our new root, `ps` will either fail or show the *host's* process list (breaking the illusion of isolation). We mount a new `proc` filesystem instance, which will only contain PIDs visible within our new PID namespace.

### `my-runc` Architecture and Execution Flow

`my-runc` is designed with a parent-child process model to achieve containerization.

#### The `run` Command (Parent Process)

When you execute `my-runc run <command> [args...]`, the initial `my-runc` process acts as the **parent**. Its primary responsibility is to:

1.  Parse the user's command.
2.  Create a new child process with all the desired Linux namespaces enabled.
3.  Set up UID/GID mappings for the user namespace.
4.  Wait for the child process to complete.

**UID/GID Mapping Explained:**

One of the most complex parts of User Namespaces is mapping. We want the user to be `root` (UID 0) inside the container, but a safe, unprivileged user (like UID 1000) outside.

In `run_linux.go`, we define `UidMappings` and `GidMappings` in `syscall.SysProcAttr`:

```go
UidMappings: []syscall.SysProcIDMap{
    {
        ContainerID: 0,           // The UID inside the container (root)
        HostID:      os.Getuid(), // The UID on the host (current user, e.g., 1000)
        Size:        1,           // Map only this one ID
    },
},
```

*   **ContainerID: 0:** This tells the kernel "Inside the new namespace, this user is 0 (root)."
*   **HostID: os.Getuid():** This tells the kernel "On the host, this maps to the actual user running the program."
*   **Result:** When the child process starts, it thinks it is running as root. It can perform operations that require root *within the bounds of its namespace* (like mounting /proc), but it has no extra privileges on the actual host filesystem.

#### The `child` Command (Child Process)

After the parent process initiates the child with new namespaces, the child process starts executing the `my-runc` binary again, but this time it enters the `child` case in the `main` function's `switch` statement. This `child` process is now isolated within its own set of namespaces. Its responsibilities are:

1.  **`setupCgroups()`**: Detects v1/v2 and applies memory limits (default 100MB).
2.  **`setupRootFS()`**: Performs the bind-mount, `pivot_root`, and mounts `/proc`.
3.  **`exec`**: Replaces itself with the user's command (e.g., `bash`).

### Platform-Specific Implementation (Go Build Tags)

Go build tags (also known as build constraints) allow us to include or exclude entire files from a package during compilation based on operating system, architecture, or custom tags. This is crucial for `my-runc` because many of the Linux kernel syscalls (like `pivot_root` or `CLONE_NEW*` flags) are not available or behave differently on other operating systems (e.g., macOS, Windows).

*   **`//go:build linux`:** This tag at the top of a `.go` file means the file will *only* be compiled when the target operating system is Linux.
*   **`//go:build !linux`:** This tag means the file will be compiled for *any* operating system *except* Linux.

### Usage

To run a command inside a `my-runc` container (on a Linux system):

```bash
sudo ./my-runc run <command> [args...]
```

**Note:** `my-runc` often requires root privileges to perform syscalls like `unshare`, `pivot_root`, and cgroup manipulations.

Example: Run `hostname` inside a container. You should see a generic hostname like `my-runc-container` (if set) or the original hostname (if UTS namespace is not fully utilized) different from your host's hostname.

```bash
sudo ./my-runc run hostname
```

Example: Run a memory-intensive Python script to test cgroups. This should fail with an OOM (Out Of Memory) error if the memory limit is exceeded.

```bash
sudo ./my-runc run python -c 'import time; x = bytearray(200000000); time.sleep(1)'
```
(Requires Python to be installed in the root filesystem used by the container)

### Building

To build the `my-runc` binary, navigate to the `my-runc` directory and run:

```bash
go build -o my-runc .
```

### Testing

Comprehensive testing is crucial to ensure the correctness and robustness of `my-runc`, especially when dealing with low-level kernel interactions. We employ a strategy that combines unit tests for different platforms.

*   **Unit Tests (`run_test.go` and `run_linux_test.go`):**
    *   `run_test.go`: Contains tests that verify the behavior of `my-runc` on non-Linux systems. It specifically checks that attempts to run containers on unsupported platforms result in the expected error message and exit code.
    *   `run_linux_test.go`: Contains tests specifically for Linux systems. These tests verify:
        *   **Hostname Isolation:** By running `hostname` inside the container and comparing it to the host's hostname, ensuring the UTS namespace is effectively isolating.
        *   **PID Isolation:** Runs `ps aux` to confirm the container sees itself as PID 1 and cannot see host processes.
        *   **Filesystem Isolation:** Verifies the `pivot_root` logic by checking if the old root is accessible.
        *   **Cgroups Memory Limit:** By attempting to allocate more memory than allowed by the cgroup (using an infinite allocation loop), it verifies that the kernel's OOM killer terminates the process as expected.

To run the tests, navigate to the `my-runc` directory and execute:

```bash
sudo go test -v ./...
```

*   **On non-Linux systems (e.g., macOS):** The `run_linux_test.go` file will be automatically skipped by the Go build system due to the `//go:build linux` tag. Only `run_test.go` (and any other untagged or `!linux` tagged tests) will run, confirming the unsupported behavior.
*   **On Linux systems:** Both `run_test.go` (if present and relevant, though typically `!linux` tests would be skipped) and `run_linux_test.go` will be executed, validating the actual containerization features. (Note: The `python` command in the cgroups test must be available in the container's root filesystem).

### Running on macOS (via Lima)

Since macOS uses the Darwin kernel, it does not support Linux namespaces or Cgroups. To develop and run `my-runc` on a Mac, we recommend using `limactl` (Lima) to run a lightweight Linux VM.

1.  **Start a Linux VM:** `limactl start ubuntu`
2.  **Enter the VM:** `limactl shell ubuntu`
3.  **Copy Code:** Due to read-only filesystem limitations with shared folders, copy the project to a local directory in the VM:
    ```bash
    cp -r /path/to/my-runc ~/playground
    cd ~/playground
    ```
4.  **Build and Run:**
    ```bash
    go build -o my-runc .
    sudo ./my-runc run bash
    ```

## Part 2: `my-kube` - The Orchestrator (Work in Progress)

We are now building a simplified version of Kubernetes to manage our `my-runc` containers across multiple nodes. We call this the **Mini-Cloud Architecture**.

### The Architecture: "Manager" vs. "Worker"

To move beyond running single processes on a single machine, we will simulate a cluster using **3 Linux VMs** (managed by `limactl`).

#### 1. The Control Plane (`my-kube-server`) - The "Manager"
*   **Location:** Runs on the `master-node` VM.
*   **Role:** The Brain. It does not run user applications. It manages the state of the cluster.
*   **Component: API Server (Go):**
    *   A Go HTTP server listening on port 8080.
    *   Accepts commands like `POST /pods` to create new work.
    *   Stores the "Desired State" (e.g., "We need 2 web servers running").
*   **Component: Scheduler:**
    *   A loop that assigns "Pending" pods to available Worker Nodes based on RAM availability.

#### 2. The Worker Node (`my-kubelet`) - The "Kitchen"
*   **Location:** Runs on `worker-node-1` and `worker-node-2`.
*   **Role:** The Muscle. It executes the work assigned by the Manager.
*   **Component: Kubelet (Go):**
    *   An agent that constantly polls the API Server: *"Do you have work for me?"*
    *   When it receives a job, it calls **`my-runc`** to spin up the container.
    *   It monitors the container's health and reports back to the Master.

#### 3. The Workload (The "App")
*   **Location:** Inside the containers on Worker Nodes.
*   **Role:** The actual application useful to the user.
*   **Implementation (Python HTTP Server):**
    *   Instead of running `ls` (which exits immediately), we will run a lightweight Python Web Server (`python3 -m http.server`).
    *   **Goal:** This server will listen on a specific IP address. If we can `curl` this IP from the Master Node and get a response, we have successfully implemented **Cluster Networking**.

### The Networking Challenge

This is the most complex part of Part 2. `ls` doesn't need network, but a Web Server does.
*   **Bridge Networking:** We must implement a virtual network bridge (like `cni0`) on each worker.
*   **IP Allocation:** Every container needs a unique IP address (e.g., `10.244.1.5`) reachable from other nodes.

### Roadmap

1.  **Infrastructure:** Script to spin up 3 connected `limactl` VMs.
2.  **Networking Upgrade:** Modify `my-runc` to support network namespaces and veth pairs (Bridge Networking).
3.  **The API Server:** Build the Go server to accept Pod requests.
4.  **The Kubelet:** Build the agent to poll the server and run containers.
5.  **Integration:** Deploy a Python Web Server pod and access it via `curl` from a different node.

## Development Notes

This project is developed primarily for educational purposes. Its goal is to provide a deep understanding of containerization and orchestration concepts by building simplified versions of widely used tools. It is not intended for production use, as it lacks robustness, security hardening, and full feature sets found in production-grade runtimes and orchestrators.
