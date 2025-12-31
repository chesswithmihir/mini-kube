# Mini-Kube: The Definitive Systems Textbook

**Author:** Mihir + Gemini  
**Date:** December 28, 2025  
**Level:** Undergraduate CS to Senior Systems Engineer  
**Vision:** Demystifying the "Magic" of the Docker + Kubernetes

---

## Table of Contents

1.  [Chapter 0: Vision, Goal, and Practical Usage](#chapter-0-vision-goal-and-practical-usage)
2.  [Chapter 1: The "Pre-Flight" Fundamentals (The CS Crash Course)](#chapter-1-the-pre-flight-fundamentals-the-cs-crash-course)
    *   [1.1 Operating Systems: The Kernel vs. Userspace](#11-operating-systems-the-kernel-vs-userspace)
    *   [1.2 Networking: The Virtual Plumbing](#12-networking-the-virtual-plumbing)
    *   [1.3 Distributed Systems: The Orchestration Loop](#13-distributed-systems-the-orchestration-loop)
3.  [Chapter 2: The Container Runtime (`my-runc`)](#chapter-2-the-container-runtime-my-runc)
    *   [2.1 The Entry Point: Command Dispatching](#21-the-entry-point-command-dispatching)
    *   [2.2 The "Re-Execution" Pattern (The Fork/Exec Loop)](#22-the-re-execution-pattern-the-forkexec-loop)
    *   [2.3 Linux Namespaces: The Six Walls of Isolation](#23-linux-namespaces-the-six-walls-of-isolation)
    *   [2.4 Filesystem Isolation: `pivot_root` internals](#24-filesystem-isolation-pivot_root-internals)
    *   [2.5 Control Groups (Cgroups): Resource Metering](#25-control-groups-cgroups-resource-metering)
4.  [Chapter 3: The Networking Stack (Building a Mini-ISP)](#chapter-3-the-networking-stack-building-a-mini-isp)
    *   [3.1 Virtual Ethernet (Veth) Pairs](#31-virtual-ethernet-veth-pairs)
    *   [3.2 Layer 2 Switching: The Bridge](#32-layer-2-switching-the-bridge)
    *   [3.3 Layer 3 Routing: The Gateway](#33-layer-3-routing-the-gateway)
    *   [3.4 NAT & IP Masquerading](#34-nat--ip-masquerading)
5.  [Chapter 4: The Orchestrator (`my-kube`)](#chapter-4-the-orchestrator-my-kube)
    *   [4.1 The API Server: Concurrency & The Memory Store](#41-the-api-server-concurrency--the-memory-store)
    *   [4.2 The Scheduler: The Logic of Placement](#42-the-scheduler-the-logic-of-placement)
    *   [4.3 The Kubelet Agent: Edge-Triggered Reconciliation](#43-the-kubelet-agent-edge-triggered-reconciliation)
6.  [Chapter 5: The Debugging Chronicles (Post-Mortems)](#chapter-5-the-debugging-chronicles-post-mortems)
    *   [5.1 Case Study: The Nexthop/Subnet Failure](#51-case-study-the-nexthopsubnet-failure)
    *   [5.2 Case Study: The 8.8.8.8 Packet Loss](#52-case-study-the-8888-packet-loss)

---

## Chapter 0: Vision, Goal, and Practical Usage

### 0.1 The Goal
The objective of this project is to build a **production-principled** container orchestrator. We are stripping away the "magic" of `docker run` and `kubectl apply` to reveal the raw Linux primitives: `clone`, `pivot_root`, `veth` pairs, and `iptables`. 

By the end of this project, you will not just *use* Kubernetes; you will understand the syscalls that make it possible.

### 0.2 The Vision: Demystifying the Black Box
We believe that Kubernetes is often treated as a "mysterious cloud engine." This project's vision is to prove that K8s is simply a **Distributed System built on top of File Operations**. By representing Namespaces, Cgroups, and Pod States as files and HTTP requests, we make the complex simple.

### 0.3 Practical Usage: Running the Runtime
`my-runc` is the foundational unit. It is the tool that actually talks to the Linux Kernel.

**Building the binary:**
```bash
cd my-runc && go build -o my-runc .
```

**Executing an isolated process with Networking:**
```bash
# This command:
# 1. Spawns a new process in a private PID/Net/Mount namespace.
# 2. Automatically creates 'my-bridge0' on your host.
# 3. Injects a virtual ethernet cable into the container.
# 4. Sets up NAT so the container can reach the internet.
sudo ./my-runc run --ip 10.244.0.100 sh -c "ip addr && ping -c 1 8.8.8.8"
```

---

## Chapter 1: The "Pre-Flight" Fundamentals (The CS Crash Course)

If you are a CS major with zero systems experience, start here. We need to define the world before we can isolate it.

### 1.1 Operating Systems: The Kernel vs. Userspace

**What is the Kernel?**
The Kernel is the "God Mode" of your computer. It is the only part of the software that is allowed to talk directly to your CPU, RAM, and Hard Drive. 

**What is Userspace?**
Everything else (your Browser, Spotify, and `my-runc`) runs in "Userspace". Userspace programs are trapped in a restricted mode. If they want to do anything interesting (like write a file or send a network packet), they must ask the Kernel for permission.

**What is a System Call (Syscall)?**
A syscall is the "API" of the Kernel. When a userspace program wants the Kernel to do something, it executes a specific instruction that "traps" the CPU and hands control to the Kernel. Examples include `open()`, `read()`, and `clone()`.

**What is a Process and a PID?**
A **Process** is an instance of a running program. Every process is assigned a **PID (Process ID)**—a unique number from 1 to 32,768. 
*   PID 1 is the "Init" process (the first process started by the Kernel).
*   If you kill PID 1, the whole system shuts down.

**What is a Filesystem and "Mounting"?**
A **Filesystem** is just a data structure on your disk that organizes files into folders. 
**Mounting** is the act of "plugging" a filesystem into a specific folder in your directory tree. 
*   Example: You "mount" your USB drive to `/media/usb`. Now, when you look inside that folder, the Kernel redirected your eyes to the USB hardware.

---

### 1.2 Networking: The Virtual Plumbing

**What is an IP Address and a Subnet?**
An **IP Address** (like `10.244.0.100`) is like a house address. 
A **Subnet Mask** (like `/16` or `255.255.0.0`) tells you which part of the address is the "Neighborhood" and which part is the "House". 
*   `/16` means the first two numbers (`10.244`) are the neighborhood.
*   Everyone in the same neighborhood can talk to each other directly.

**What is a Bridge?**
A **Bridge** is a virtual "Network Switch". Imagine a power strip for network cables. You plug multiple virtual machines or containers into a bridge, and they can all "see" each other's traffic.

**What is a Veth Pair (Virtual Ethernet)?**
Imagine a physical Ethernet cable. Now imagine a *magical* Ethernet cable where you can stretch it across the boundaries of space and time. 
A **Veth Pair** has two ends. What goes in End A comes out End B, even if End B is inside an isolated container namespace.

**What is NAT (Network Address Translation)?**
Your container has a "Private" IP (`10.244.0.100`). The Internet doesn't know what that is. 
**NAT** is like an office mailroom. When you send a letter out, the mailroom replaces your "Desk Number" with the "Office Building Address". When a reply comes back, the mailroom remembers who sent the original letter and puts it back on your desk.

---

### 1.3 Distributed Systems: The Orchestration Loop

**What is a Distributed System?**
It is a collection of independent computers that appears to its users as a single coherent system.

**The "Desired State" vs. "Actual State"**
This is the core of Kubernetes.
*   **Desired State:** "I want 3 copies of Nginx running."
*   **Actual State:** "Only 2 copies are running because one crashed."
*   **The Reconciliation Loop:** A program that constantly looks at both states and says, "Oh, I need to start one more copy to make them match."

---

## Chapter 2: The Container Runtime (`my-runc`)

This is the code that lives in the `my-runc/` directory.

### 2.1 The Entry Point: Command Dispatching

**File:** `my-runc/main.go`

Our `main()` function uses a `switch` statement to handle the "Parent" vs. "Child" logic. This is the initial dispatch point when you run the `my-runc` executable.

```go
// my-runc/main.go (simplified)
func main() {
    // ... argument parsing ...
    command := os.Args[1] // e.g., "run", "child"
    
    switch command {
    case "run":
        // The Parent: This case is executed when you run `sudo ./my-runc run ...`
        // It's responsible for setting up the initial environment and re-executing itself
        // in "child" mode to perform the actual container setup.
        ip := os.Args[3] // Extract IP address if provided
        commandToRun := os.Args[4:] // The command to run inside the container
        run(commandToRun, ip) 
    case "child":
        // The Child: This case is executed when my-runc calls itself with the "child" argument.
        // It's now running inside the newly created namespaces and will set up cgroups,
        // the root filesystem, and finally execute the user's command.
        setupCgroups()
        setupRootFS("/") // Setup the isolated filesystem
        
        // After setup, the child process replaces itself with the user's command.
        // os.Args = os.Args[2:] adjusts arguments so the first argument is the command itself.
        // The Run() here is usually a wrapper around syscall.Exec.
        // This is the final `exec` in the chain.
        os.Args = os.Args[2:] 
        exec.Command(os.Args[0], os.Args[1:]...).Run() 
    }
}
```

### 2.2 The "Re-Execution" Pattern (The Fork/Exec Loop)

**File:** `my-runc/run_linux.go`

This section details the "re-execution" pattern, which is crucial for setting up the container environment. It's important to note that this pattern primarily relies on the `exec` system call, not `fork` followed by `exec` in the traditional sense where the parent process continues to run.

**How `exec` works here:**

1.  **Parent Initiates `exec`:** The `run()` function in `run_linux.go` prepares the environment for the new process. It configures `syscall.SysProcAttr` with the desired namespace flags. Then, it calls `exec.Command("/proc/self/exe", "child", ...)`.
    *   `/proc/self/exe`: This is a special symbolic link in Linux that always points to the executable file of the current process. When `my-runc` executes itself, this path correctly refers back to the `my-runc` binary.
    *   `"child"`: This is an argument passed to the re-executed process, signaling it to enter the "child" mode where it performs the container setup.
    *   The rest of the arguments (like `--ip` and the user's command) are also passed along.

    ```go
    // my-runc/run_linux.go (within the run() function)
    func run(commandToRun []string, ip string) {
        log.Printf("Running command: %s with IP: %s", commandToRun, ip)

        // 1. Create Synchronization Pipe
        // This pipe is used to synchronize the parent and child processes.
        // The parent will block until network setup is done, then signal the child to proceed.
        r, w, err := os.Pipe()
        if err != nil {
            log.Fatalf("Failed to create pipe: %v", err)
        }

        // Re-execute my-runc itself with the "child" command.
        // This process will *replace* the current 'run' process.
        cmd := exec.Command("/proc/self/exe", append([]string{"child"}, commandToRun...)...)
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        
        // 4. Configure SysProcAttr (Linux-specific process attributes)
        // This is where we define the namespaces for the *new* process.
        cmd.SysProcAttr = &syscall.SysProcAttr{
            Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET | syscall.CLONE_NEWUSER,
            UidMappings: []syscall.SysProcIDMap{
                {
                    ContainerID: 0, // Inside container, uid is 0 (root)
                    HostID:      os.Getuid(), // Outside, it maps to the current host user's UID
                    Size:        1,
                },
            },
            GidMappings: []syscall.SysProcIDMap{
                {
                    ContainerID: 0, // Inside container, gid is 0 (root)
                    HostID:      os.Getgid(), // Outside, it maps to the current host user's GID
                    Size:        1,
                },
            },
        }

        // Pass the read-end of the pipe to the child process.
        // It will be available as file descriptor 3 in the child.
        cmd.ExtraFiles = []*os.File{r}

        // 2. Start the Child (Fork/Clone + Exec)
        // cmd.Start() internally calls clone() with the configured SysProcAttr flags
        // and then execve() for the new process.
        // The child will start but block reading from FD 3, waiting for the parent's signal.
        if err := cmd.Start(); err != nil {
            log.Fatalf("Failed to run container: %v", err)
        }

        // Close parent's copy of the read-end (r) as it's no longer needed in the parent.
        r.Close()

        // 3. Setup Network (if requested) - this happens in the parent process.
        // The network setup often involves moving one end of a veth pair into the child's
        // newly created network namespace.
        if ip != "" {
            if err := setupNetwork(cmd.Process.Pid, ip+"/16", "my-bridge0", "10.244.0.1/16"); err != nil {
                log.Printf("Setup network failed: %v", err)
                cmd.Process.Kill()
                os.Exit(1)
            }
        }

        // 4. Signal Child to Continue
        // Write "OK" to the pipe. This unblocks the child process.
        // The child can now proceed with its internal setup (cgroups, rootfs, etc.).
        w.Write([]byte("OK"))
        w.Close() // Close the write-end of the pipe.

        // 5. Wait for Child to Finish
        // The parent waits for the container process (which is now the user's command) to complete.
        if err := cmd.Wait(); err != nil {
            log.Fatalf("Container process failed: %v", err)
        }
    }
    ```

2.  **The Mystery of `/proc/self/exe`**: As mentioned, this path ensures that `my-runc` is calling a copy of itself. This is essential because the Go runtime needs to be active within the new namespaces to perform their setup (like mounting `/proc` inside the new PID namespace).

3.  **Why?**: The `exec` system call is used to *replace* the current process's image with a new one. This means the original parent `my-runc` process is entirely replaced by the new `my-runc` process running in "child" mode. This "child" `my-runc` process then proceeds to set up the container's namespaces, root filesystem, and cgroups. Once setup is complete, it uses `exec` *again* (within its `main.go`'s "child" case logic) to finally replace itself with the user's desired command (e.g., `bash`). This chain of `exec` calls ensures that the Go setup code runs within the correct isolated environment before the final command takes over.

### 2.3 Linux Namespaces: The Six Walls of Isolation

**File:** `my-runc/run_linux.go` (primarily where `SysProcAttr` is configured) and `my-runc/namespace_linux.go` (for functions like `setupRootFS`, `setupCgroups`).

In the `run()` function within `my-runc/run_linux.go`, we configure the `syscall.SysProcAttr` struct. This is where we instruct the Linux Kernel on how to create the new, isolated environment for the child process.

```go
// my-runc/run_linux.go (within the run() function, part of cmd.SysProcAttr definition)
cmd.SysProcAttr = &syscall.SysProcAttr{
    // Cloneflags: These flags tell the kernel to create new namespaces for the child process.
    // Each CLONE_NEW* flag isolates a different aspect of the process environment.
    Cloneflags: syscall.CLONE_NEWPID | // Create a new Process ID namespace
                syscall.CLONE_NEWNS | // Create a new Mount namespace
                syscall.CLONE_NEWUTS | // Create a new UTS (hostname) namespace
                syscall.CLONE_NEWIPC | // Create a new IPC namespace
                syscall.CLONE_NEWNET | // Create a new Network namespace
                syscall.CLONE_NEWUSER, // Create a new User namespace

    // UidMappings and GidMappings: These map user and group IDs between the host and the container.
    // This is crucial for user namespace isolation (CLONE_NEWUSER).
    // It allows a process to be 'root' (UID 0) inside the container, but map to an unprivileged
    // user ID on the host, enhancing security.
    UidMappings: []syscall.SysProcIDMap{
        {
            ContainerID: 0, // Inside container, uid is 0 (root)
            HostID:      os.Getuid(), // Outside, it maps to the current host user's UID
            Size:        1,           // Map only 1 UID
        },
    },
    GidMappings: []syscall.SysProcIDMap{
        {
            ContainerID: 0, // Inside container, gid is 0 (root)
            HostID:      os.Getgid(), // Outside, it maps to the current host user's GID
            Size:        1,           // Map only 1 GID
        },
    },
}
```

*   **`CLONE_NEWPID` (The Matrix)**:
    *   **Metaphor:** Every process gets a new ID starting at 1.
    *   **Code Detail:** The `CLONE_NEWPID` flag ensures that the first process launched in the new container (which is our "child" `my-runc` process, then replaced by the user's command) will see itself as PID 1 within its own namespace.
    *   **Depth:** The Kernel maintains a translation table. Inside, you are PID 1. Outside, you are PID 4502. If you try to kill PID 2 (the host's second process), the Kernel says "PID 2? Never heard of her."

*   **`CLONE_NEWNET` (The Silence)**:
    *   **Metaphor:** Cutting the phone lines.
    *   **Code Detail:** With `CLONE_NEWNET`, the child process starts with an empty network stack. The `setupNetwork` function in the parent (`run_linux.go`), using the child's PID, is responsible for configuring network devices (like moving a `veth` end) *into* this new network namespace. The child then brings up its own `lo` interface.
    *   **Depth:** The process has no network devices until they are manually set up. It cannot even talk to `localhost` until the `lo` interface is brought up.

*   **`CLONE_NEWUSER` (The Fake Identity)**:
    *   **Metaphor:** You are a King in your room, but a Peasant in the hallway.
    *   **Code Detail:** The `UidMappings` and `GidMappings` within `SysProcAttr` define how user/group IDs inside the new user namespace map to IDs on the host. This allows the container's root user (UID 0) to be mapped to an unprivileged user on the host, preventing the container's root from having root privileges outside its namespace.
    *   **Depth:** Maps Container-UID 0 (Root) to Host-UID 1000 (You). You can run `apt install` inside because you are "root", but you cannot delete `/etc/shadow` on the host because the host sees you as a regular user.

### 2.4 Filesystem Isolation: `pivot_root` internals

**File:** `my-runc/namespace_linux.go` -> `setupRootFS()`

This function is responsible for establishing the container's root filesystem using `pivot_root`, which is preferred over `chroot` because it provides stronger isolation. This logic runs in the "child" `my-runc` process *after* it has been launched into its new namespaces.

**The Step-by-Step Logic within `setupRootFS()`:**

```go
// my-runc/namespace_linux.go
func setupRootFS(rootfs string) error {
	log.Println("Setting up root filesystem...")

	// 1. Create a location for the new root
	// We use a temporary directory to act as the new root mount point.
	// This directory (`/tmp/my-runc-root`) will temporarily hold the new rootfs before pivot_root.
	newRoot := "/tmp/my-runc-root"
	if err := os.MkdirAll(newRoot, 0755); err != nil {
		return fmt.Errorf("failed to create new root dir: %w", err)
	}

	// 2. Bind mount the desired rootfs (e.g. "/") to the new location.
	// `rootfs` is typically the path to the container's image (e.g., a busybox directory).
	// `syscall.MS_BIND` makes a bind mount: the contents of `rootfs` now appear at `newRoot`.
	// `syscall.MS_REC` makes it recursive, binding all submounts.
	// This makes 'newRoot' a mount point with the content of 'rootfs'.
	if err := syscall.Mount(rootfs, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to bind mount rootfs to new root: %w", err)
	}

	// 3. Make the new root's mount point private to this namespace
	// `syscall.MS_PRIVATE` ensures that any mount/unmount events within this mount point
	// are not propagated to other mount namespaces (e.g., the host).
	if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to make new root private: %w", err)
	}

	// 4. Create directory for the old root inside the new root
	// `putOld` is a temporary directory within `newRoot` where the *original* rootfs
	// will be mounted after the `pivot_root` call. This is required by `pivot_root`.
	putOld := filepath.Join(newRoot, ".pivot_root")
	if err := os.Mkdir(putOld, 0777); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create put_old directory: %w", err)
	}

	// 5. Pivot Root
	// `syscall.PivotRoot(newRoot, putOld)`: This is the atomic swap.
	// The `newRoot` (`/tmp/my-runc-root`) becomes the new root filesystem (`/`) for this process.
	// The *original* root filesystem is moved and mounted at `putOld` (`/.pivot_root`).
	if err := syscall.PivotRoot(newRoot, putOld); err != nil {
		return fmt.Errorf("failed to pivot_root: %w", err)
	}

	// 6. Change the current working directory to the new root ("/")
	// After `pivot_root`, the current working directory might still be pointing
	// to a location in the old root, so we change to `/` (the new root).
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("failed to change directory to new root: %w", err)
	}

	// 7. Unmount the old root
	// The old root, now at `/.pivot_root`, needs to be unmounted to completely
	// sever the connection to the host's original filesystem.
	// `syscall.MNT_DETACH` allows unmounting even if the filesystem is busy.
	if err := syscall.Unmount("/.pivot_root", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("failed to unmount old root: %w", err)
	}

	// 8. Remove the temporary directory `/.pivot_root`
	// This removes the now empty directory that held the old root mount point.
	if err := os.Remove("/.pivot_root"); err != nil {
		return fmt.Errorf("failed to remove put_old directory: %w", err)
	}

	// 9. Mount proc filesystem
	// The `/proc` filesystem is crucial for processes to interact with the kernel
	// and view process information (e.g., `ps`, `top`). Since we have a new PID
	// namespace, we need to mount a new `/proc` specifically for this container.
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("failed to mount proc: %w", err)
	}

	log.Println("Root filesystem setup complete")
	return nil
}
```

*   **The old host filesystem is now gone** from the container's perspective. If the process tries to `cd ../../../`, it will just navigate within the new root filesystem, as it's physically impossible to see the host's files.

### 2.5 Control Groups (Cgroups): Resource Metering

**File:** `my-runc/namespace_linux.go` -> `setupCgroups()`

Namespaces are about **Isolation**. Cgroups are about **Resource Limits**. This function configures resource constraints for the container. This logic runs in the "child" `my-runc` process, configuring limits for itself and any child processes it spawns (i.e., the user's command).

**How we talk to the Kernel via the Cgroup Filesystem:**
The Linux Kernel exposes Cgroups as a virtual filesystem, typically mounted at `/sys/fs/cgroup`. We interact with it by creating directories and writing values to control files.

```go
// my-runc/namespace_linux.go
func setupCgroups() error {
	log.Println("Setting up cgroups...")

	cgroupPath := "/sys/fs/cgroup"
	var memPath string
	var limitFile string
	
	// Dynamically determine Cgroups version (v1 or v2)
	// Cgroups v1 often has controllers (like 'memory') mounted separately.
	// Cgroups v2 uses a unified hierarchy.
	if _, err := os.Stat(filepath.Join(cgroupPath, "memory")); err == nil {
		// Cgroups v1 detected: memory controller is mounted at /sys/fs/cgroup/memory
		log.Println("Detected Cgroups v1")
		memPath = filepath.Join(cgroupPath, "memory", "my-container") // Path for our container's memory cgroup
		limitFile = "memory.limit_in_bytes" // Cgroups v1 uses this file for memory limit
	} else {
		// Cgroups v2 detected: unified hierarchy, control files are directly in our cgroup dir.
		log.Println("Detected Cgroups v2")
		memPath = filepath.Join(cgroupPath, "my-container") // Path for our container's cgroup
		limitFile = "memory.max" // Cgroups v2 uses 'memory.max' for memory limit
	}

	// Create the cgroup directory for this container.
	// If it already exists, it proceeds without error.
	if err := os.Mkdir(memPath, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create cgroup directory: %w", err)
	}

	// Move the current process (the "child" my-runc process) into the new cgroup.
	// This means all processes spawned by this "child" process (including the user's command)
	// will inherit these cgroup limits.
	pid := os.Getpid()
	procsFile := filepath.Join(memPath, "cgroup.procs") // File to list processes in the cgroup (v1)
	if _, err := os.Stat(procsFile); os.IsNotExist(err) { // If cgroup.procs doesn't exist (e.g., v2)
		procsFile = filepath.Join(memPath, "cgroup.threads") // Use cgroup.threads for v2
	}
	
	if err := os.WriteFile(procsFile, []byte(fmt.Sprintf("%d", pid)), 0700); err != nil {
		return fmt.Errorf("failed to write to cgroup.procs/threads: %w", err)
	}

	// Set a memory limit of 100MB for the container.
	// This writes the limit to the appropriate file (memory.limit_in_bytes for v1, memory.max for v2).
	memoryLimitBytes := "100000000" // 100MB
	limitFilePath := filepath.Join(memPath, limitFile)
	if err := os.WriteFile(limitFilePath, []byte(memoryLimitBytes), 0700); err != nil {
		return fmt.Errorf("failed to write to memory limit file: %w", err)
	}

	// Additional cgroup configurations could be added here for CPU, I/O, etc.
	// For example, setting CPU shares:
	// if err := os.WriteFile(filepath.Join(memPath, "cpu.shares"), []byte("1024"), 0700); err != nil {
	//     log.Printf("Failed to set CPU shares: %v", err)
	// }

	log.Println("Cgroups setup complete")
	return nil
}
```

*   **How we talk to the Kernel:** The Kernel exposes Cgroups as a filesystem at `/sys/fs/cgroup`.
*   **1. Create Directory:** We `mkdir` a folder there (e.g., `/sys/fs/cgroup/memory/my-container` for v1 or `/sys/fs/cgroup/my-container` for v2). This creates the control group for our container.
*   **2. Add PID to `cgroup.procs` (or `cgroup.threads`):** We write the container's PID into the relevant file. This associates the process with the defined cgroup. Any processes added here (and their children) will be subject to the cgroup's limits.
*   **3. Set Limits:** We write a value (e.g., `100000000` for 100MB) into a control file like `memory.limit_in_bytes` (v1) or `memory.max` (v2).
*   **The Enforcement:** The Kernel now monitors the process. The moment it attempts to exceed the allocated 100MB of RAM, the Kernel triggers the **OOM Killer** (Out-Of-Memory Killer) and sends a `SIGKILL` signal to terminate the process, enforcing the limit.

---

## Chapter 3: The Networking Stack (Building a Mini-ISP)

### 3.1 Virtual Ethernet (Veth) Pairs

**File:** `my-runc/network_linux.go`

In `setupNetwork()`, we perform the "magical cable" trick.

```go
exec.Command("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethContainer)
```

1.  We create the two ends.
2.  We move the `vethContainer` end into the child's PID namespace:
    `ip link set veth-c123 netns 123`
3.  Now, the host has one end, and the container has the other. They are physically connected by a virtual wire.

### 3.2 Layer 2 Switching: The Bridge

We created `my-bridge0`.
*   **Switching Logic:** When the container sends a packet, it hits the bridge. The bridge looks at the MAC address and says "Oh, this is for the Host" or "This is for Container B".
*   **The IP:** We give the bridge the IP `10.244.0.1`. This is the "Gateway". Every container knows: "If I want to talk to the internet, I send my packet to `10.244.0.1`."

### 3.4 NAT & IP Masquerading

**The Debugging Revelation:**
Even with a cable and a bridge, the container couldn't talk to Google. Why?
Because Google's routers don't know where `10.244.0.100` is. That's a "Private" address.

**The Fix:**
We used `iptables` to perform **Masquerading**.
*   When a packet from `10.244.0.100` leaves the host, the host Kernel **erases** that address and writes **its own** public address instead.
*   It remembers: "Packet ID #445 was actually from the container."
*   When Google replies, the host Kernel checks its table, sees #445, and puts the container's address back on the packet before handing it back to the bridge.

---

## Chapter 4: The Orchestrator (`my-kube`)

### 4.1 The API Server: Concurrency & The Memory Store

**File:** `my-kube/pkg/server/store.go`

Our API server is an HTTP server that stores the "State of the World".

**The Mutex Pattern:**
Because multiple agents might call the server at the same time, we use a `sync.RWMutex`.
*   **The Problem:** If two programs write to a Go `map` at the same time, the program crashes with a "fatal error".
*   **The Solution:** Before writing, we call `mu.Lock()`. This makes every other program wait in line. Once we are done, we call `Unlock()`.

### 4.2 The Scheduler: The Logic of Placement

**File:** `my-kube/server/main.go` -> `runScheduler()`

The Scheduler is like a dispatcher at a taxi company. 
1.  It looks at "Pending Pods" (People waiting for a taxi).
2.  It looks at "Nodes" (Available taxis).
3.  It "Binds" them: "Pod A, you go to Node 1."

It does this in a loop every 5 seconds. This is the "Brain" of the orchestrator.

### 4.3 The Kubelet Agent: Edge-Triggered Reconciliation

**File:** `my-kube/agent/agent.go`

The Kubelet runs on every worker. It is the most "diligent" part of the system.
**The Loop:**
1.  "Hey Server, what should I be running?"
2.  "Server says: Pod A and Pod B."
3.  "Kubelet checks: I'm only running Pod A."
4.  "Kubelet concludes: I must start Pod B."
5.  "Kubelet calls `my-runc run --ip ... Pod B`."

This loop is **Self-Healing**. If you manually kill Pod B, the Kubelet will notice it's gone in the next 5 seconds and restart it.

---

## Chapter 5: The Debugging Chronicles (Post-Mortems)

### 5.1 Case Study: The Nexthop/Subnet Failure

**The Symptom:**
`nsenter command [ip route add default via 10.244.0.1] failed: Error: Nexthop has invalid gateway.`

**The CS Explanation:**
A "Nexthop" is the next router in the chain. Linux refused to add it because we told the container its IP was `10.244.0.100` with a `/32` mask.
*   `/32` means "My network is only 1 person wide (me)."
*   The container thought the gateway `10.244.0.1` was in a different country.
*   **The Fix:** We used `/16`. Now the container knows `10.244.0.1` is its neighbor in the same "neighborhood".

### 5.2 Case Study: The 8.8.8.8 Packet Loss

**The Symptom:**
`PING 8.8.8.8: 1 packets transmitted, 0 received, 100% packet loss`

**The CS Explanation:**
The packet went out, but it couldn't find its way back. This is because "Private IPs" (10.x.x.x) are invisible to the public internet.
*   **The Fix:** We enabled NAT Masquerading. This turned the Host into a "Proxy" that handles the public identity for all containers.

---

## Final Word to the CS Student

You have just read the blueprint of the modern cloud. Every time you deploy an app to AWS, Google Cloud, or Azure, these exact syscalls (`clone`, `pivot_root`, `iptables`) are happening billions of times per second. 

**Containers aren't magic. They are just clever lies told by the Kernel.**