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
3.  [Chapter 2: The Container Runtime (`my-runc`) - A Step-by-Step Flow](#chapter-2-the-container-runtime-my-runc---a-step-by-step-flow)
    *   [Introduction: The Journey of a Container Command](#introduction-the-journey-of-a-container-command)
    *   [2.1 Initial Invocation: The User's Command (`my-runc/main.go`)](#21-initial-invocation-the-users-command-my-runcmaingo)
    *   [2.2 Phase 1: The Parent Process - Setting the Stage (`my-runc/run_linux.go`)](#22-phase-1-the-parent-process---setting-the-stage-my-runcrun_linuxgo)
    *   [2.3 Phase 2: The Child Process - Internal Container Setup (`my-runc/main.go` and `my-runc/namespace_linux.go`)](#23-phase-2-the-child-process---internal-container-setup-my-runcmaingo-and-my-runcnamespace_linuxgo)
    *   [2.4 Linux Namespaces: The Walls of Isolation](#24-linux-namespaces-the-walls-of-isolation)
    *   [2.5 Filesystem Isolation: `pivot_root` Internals](#25-filesystem-isolation-pivot_root-internals)
    *   [2.6 Control Groups (Cgroups): Resource Metering](#26-control-groups-cgroups-resource-metering)
4.  [Chapter 3: The Networking Stack (Building a Mini-ISP)](#chapter-3-the-networking-stack-building-a-mini-isp)
    *   [3.1 Virtual Ethernet (Veth) Pairs](#31-virtual-ethernet-veth-pairs)
    *   [3.2 Layer 2 Switching: The Bridge](#32-layer-2-switching-the-bridge)
    *   [3.3 Layer 3 Routing: The Gateway](#33-layer-3-routing-the-gateway)
    *   [3.4 NAT & IP Masquerading](#34-nat--ip-masquerading)
5.  [Chapter 4: The Orchestrator (`my-kube`)](#chapter-4-the-orchestrator-my-kube)
    *   [4.1 The API Server: Concurrency & The Memory Store](#41-the-api-server-concurrency--the-memory-store)
    *   [4.2 The Scheduler: The Logic of Placement](#42-the-scheduler-the-logic-of-placement)
    *   [4.3 The Kubelet Agent: Edge-Triggered Reconciliation](#43-the-kubelet-agent-edge-triggered-reconciliation)
6.  [Chapter 5: Infrastructure & Testing (The Engineering Backbone)](#chapter-5-infrastructure--testing-the-engineering-backbone)
    *   [5.1 The Build System: Makefiles & Cross-Compilation](#51-the-build-system-makefiles--cross-compilation)
    *   [5.2 End-to-End Testing: Mocking the Runtime](#52-end-to-end-testing-mocking-the-runtime)
    *   [5.3 Virtualization Architecture: The "Split-Brain" Network Problem](#53-virtualization-architecture-the-split-brain-network-problem)
7.  [Chapter 6: The Debugging Chronicles (Post-Mortems)](#chapter-6-the-debugging-chronicles-post-mortems)
    *   [6.1 Case Study: The Nexthop/Subnet Failure](#61-case-study-the-nexthopsubnet-failure)
    *   [6.2 Case Study: The 8.8.8.8 Packet Loss](#62-case-study-the-8888-packet-loss)

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
make my-runc
```

**Executing an isolated process with Networking:**
```bash
# This command:
# 1. Spawns a new process in a private PID/Net/Mount namespace.
# 2. Automatically creates 'my-bridge0' on your host.
# 3. Injects a virtual ethernet cable into the container.
# 4. Sets up NAT so the container can reach the internet.
sudo ./bin/my-runc run --ip 10.244.0.100 sh -c "ip addr && ping -c 1 8.8.8.8"
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

## Chapter 2: The Container Runtime (`my-runc`) - A Step-by-Step Flow

This chapter details the entire lifecycle of launching a container using `my-runc`, from the initial command line invocation to the execution of the user's process within its isolated environment. We will trace the flow through distinct phases involving the "Parent" and "Child" processes, clarifying the crucial role of `exec` system calls and inter-process communication.

### Introduction: The Journey of a Container Command

When you type `sudo ./my-runc run --ip 10.244.0.100 sh -c "ip addr && ping -c 1 8.8.8.8"`, you initiate a complex dance between several processes and the Linux kernel. Here's the high-level sequence:

1.  **User Invokes `my-runc` (Initial Process):** This is the first `my-runc` instance, which we'll call the **Parent Process**. Its main job is to prepare for the container and launch a specialized child.
2.  **Parent Launches Child (`clone()` + First `exec`):** The Parent Process creates a *new* process, explicitly asking the kernel to create it within specific Linux Namespaces (PID, Network, Mount, etc.). This new process then immediately `exec`s the `my-runc` binary itself, but now running in "child" mode. This is the **Child Process**. Crucially, the *original Parent Process continues to run* at this stage.
3.  **Child Performs Internal Setup:** The Child Process, now running inside the new namespaces, performs essential internal configuration like setting resource limits (Cgroups) and establishing its isolated root filesystem (`pivot_root`).
4.  **Parent Completes External Setup:** While the Child is paused, the Parent Process finishes any external configuration that requires the Child's newly created namespaces (e.g., setting up the network for the container).
5.  **Synchronization and Handover:** The Parent signals the Child that external setup is complete.
6.  **Child Launches User Command (Second `exec`):** The Child Process then `exec`s the user's requested command (e.g., `sh -c "ip addr..."`), *replacing itself* entirely with that command. This is the final process running inside the container.
7.  **Parent Waits:** The Parent Process waits for the user's command to complete, then cleans up.

This journey involves two distinct `exec` calls and a synchronized interaction to ensure a robust and isolated container environment.

### 2.1 Initial Invocation: The User's Command (`my-runc/main.go`)

**File:** `my-runc/main.go`

When you execute `my-runc` from the command line, the `main()` function is the first entry point. It parses the command-line arguments and dispatches to different internal functions based on the subcommand provided.

```go
// my-runc/main.go (simplified excerpt)
package main

import (
	"log"
	"os"
	"os/exec"
    "io" // Used by the 'child' case
)

func main() {
    // Basic argument validation
	if len(os.Args) < 2 {
		log.Fatal("Usage: my-runc <command> [args...]")
	}

	command := os.Args[1]
	log.Printf("my-runc command: %s", command)

	switch command {
	case "run":
        // This 'run' case is for the initial user invocation.
        // It prepares arguments and calls the `run` function (defined in run_linux.go).
		args := os.Args[2:]
		var ip string // Extracts optional IP for network setup
		if len(args) > 0 && args[0] == "--ip" {
			if len(args) < 3 {
				log.Fatal("Usage: my-runc run --ip <ip> <command>")
			}
			ip = args[1]
			args = args[2:]
		} else {
			if len(args) < 1 {
				log.Fatal("Usage: my-runc run <command>")
			}
		}
		run(args, ip) // Hand off to the 'run' function for parent logic

	case "child":
        // This 'child' case is NOT called directly by the user.
        // It's called when the *parent* `my-runc` process re-executes itself
        // to create the actual container environment.
        // Its detailed flow is covered in Section 2.3.
		if len(os.Args) < 3 { /* ... */ }
		commandToRun := os.Args[2:]
		log.Printf("Running command in child: %s", commandToRun)

		// 1. Setup cgroups for resource limiting
		if err := setupCgroups(); err != nil { /* ... */ }

		// 2. Setup root filesystem for isolation
		if err := setupRootFS("/"); err != nil { /* ... */ }

		// 3. WAIT FOR NETWORK SIGNAL (Synchronization with Parent)
		pipe := os.NewFile(3, "pipe") /* ... */
		log.Println("Waiting for network setup signal from parent...")
		if _, err := io.ReadAll(pipe); err != nil { /* ... */ }
		pipe.Close()
		log.Println("Network setup signal received. Proceeding to execute command.")

		// 4. Execute the user's command (This will be the SECOND `exec` in the overall flow)
		cmd := exec.Command(commandToRun[0], commandToRun[1:]...) /* ... */
		if err := cmd.Run(); err != nil { /* ... */ }

	case "spec":
        // This command (placeholder) would generate a container configuration file (e.g., an OCI runtime specification).
		log.Println("Generating container specification (placeholder)...")

	case "version":
        // This command displays the current version of the my-runc binary.
		log.Println("my-runc version 0.1.0")

	default:
		log.Printf("Unknown command: %s", command)
		log.Println("Available commands: run, child, spec, version")
		os.Exit(1)
	}
}
```

### 2.2 Phase 1: The Parent Process - Setting the Stage (`my-runc/run_linux.go`)

**File:** `my-runc/run_linux.go`

After `my-runc run` is invoked, the `main()` function calls the `run()` function. This `run()` function is the **Parent Process** responsible for orchestrating the initial steps of container creation. It prepares the environment and launches the specialized "child" `my-runc` process in its own isolated namespaces.

**The Parent Process Flow:**

1.  **Creating a Synchronization Pipe:**
    The parent first creates an anonymous pipe (`r`, `w`). This pipe is a critical inter-process communication (IPC) mechanism used to synchronize the parent's external setup actions with the child's internal workflow.
    ```go
    // my-runc/run_linux.go (within run() function)
    r, w, err := os.Pipe()
    if err != nil {
        log.Fatalf("Failed to create pipe: %v", err)
    }
    ```
    *Detail*: `r` is the read-end, `w` is the write-end. The child will inherit `r` and block on it, while the parent holds `w` and writes to it when ready.

2.  **Configuring Child Process Attributes (`syscall.SysProcAttr`):**
    The parent prepares a command (`cmd`) to re-execute `my-runc` itself as a child. Crucially, it configures `cmd.SysProcAttr` with specific flags that tell the Linux kernel *how* to create the new child process.
    ```go
    // my-runc/run_linux.go (within run() function)
    cmd := exec.Command("/proc/self/exe", append([]string{"child"}, commandToRun...)...)
    // ...
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET | syscall.CLONE_NEWUSER,
        UidMappings: []syscall.SysProcIDMap{ /* ... */ },
        GidMappings: []syscall.SysProcIDMap{ /* ... */ },
    }
    ```
    *Detail*: The `Cloneflags` specify which new namespaces (PID, Mount, UTS, IPC, Network, User) the child process should be created within. `UidMappings` and `GidMappings` configure user ID remapping for security. (See Section 2.4 for more on Namespaces).

3.  **Passing the Pipe's Read-End to the Child:**
    The parent passes the read-end (`r`) of the synchronization pipe to the child process using `cmd.ExtraFiles`. This makes `r` available to the child as file descriptor 3.
    ```go
    // my-runc/run_linux.go (within run() function)
    cmd.ExtraFiles = []*os.File{r}
    ```
    *Detail*: The child will use this to receive signals from the parent.

4.  **Launching the Child Process (First `exec`):**
    The parent calls `cmd.Start()`. This is the first significant `exec` transition.
    ```go
    // my-runc/run_linux.go (within run() function)
    if err := cmd.Start(); err != nil {
        log.Fatalf("Failed to run container: %v", err)
    }
    ```
    *Detail*: `cmd.Start()` performs the `clone()` system call with the `SysProcAttr` flags, creating a *new process* that immediately `exec`s the `my-runc` binary (in "child" mode) into its new namespaces. **The original Parent Process `my-runc run` *continues to run* after this step; it is not replaced yet.**

5.  **Parent Closes its Read-End:**
    Immediately after launching the child, the parent closes its copy of the pipe's read-end (`r`). This is good practice to avoid resource leaks.
    ```go
    // my-runc/run_linux.go (within run() function)
    r.Close()
    ```

6.  **Parent Performs External Setup (e.g., Network Configuration):**
    With the child process now existing in its new namespaces, the parent can perform external setup tasks that require interaction with these namespaces. Network configuration, for instance, often involves moving virtual network interfaces *into* the child's network namespace, an operation that must be done by the parent (or a privileged process).
    ```go
    // my-runc/run_linux.go (within run() function)
    if ip != "" { // If an IP was specified, configure the network
        // `setupNetwork` uses the child's PID to move network devices into its new namespace.
        if err := setupNetwork(cmd.Process.Pid, ip+"/16", "my-bridge0", "10.244.0.1/16"); err != nil {
            log.Printf("Setup network failed: %v", err)
            cmd.Process.Kill() // Kill child if network setup fails
            os.Exit(1)
        }
    }
    ```
    *Detail*: At this point, the child is blocked on the pipe, waiting.

7.  **Parent Signals Child to Proceed:**
    Once all external setup (like network configuration) is complete, the parent writes a byte to the pipe's write-end (`w`). This unblocks the child process. The parent then closes its write-end (`w`).
    ```go
    // my-runc/run_linux.go (within run() function)
    w.Write([]byte("OK"))
    w.Close()
    ```
    *Detail*: This is the synchronization point.

8.  **Parent Waits for Child's Completion:**
    Finally, the parent calls `cmd.Wait()`, which pauses the parent until the child process (which, as we'll see, will eventually become the user's command) exits.
    ```go
    // my-runc/run_linux.go (within run() function)
    if err := cmd.Wait(); err != nil {
        log.Fatalf("Container process failed: %v", err)
    }
    ```
    *Detail*: This ensures the `my-runc` command itself doesn't exit until the container has finished its work.

### 2.3 Phase 2: The Child Process - Internal Container Setup (`my-runc/main.go` and `my-runc/namespace_linux.go`)

**File:** `my-runc/main.go` and `my-runc/namespace_linux.go`

The "child" `my-runc` process begins its execution immediately after the `cmd.Start()` call by the parent. It runs the `main()` function's `child` case, already within its newly created, isolated namespaces. Its role is to finalize the container's environment and then hand control to the user's specified command.

**The Child Process Flow:**

1.  **Child `my-runc` Starts in New Namespaces:**
    The process begins executing the `child` case in `main.go`. At this point, it is already operating within its own dedicated PID, Mount, Network, UTS, IPC, and User namespaces, as requested by the parent's `Cloneflags`. It has also inherited file descriptor 3, which is the read-end of the synchronization pipe.
    ```go
    // my-runc/main.go (within case "child")
    // ... (argument parsing) ...
    log.Printf("Running command in child: %s", commandToRun)
    ```

2.  **Internal Resource Limit Setup (`setupCgroups()`):**
    The child first calls `setupCgroups()`. This function configures resource constraints (like memory limits) for the container. It does this by interacting with the cgroup filesystem, which is exposed by the kernel.
    ```go
    // my-runc/main.go (within case "child")
    if err := setupCgroups(); err != nil {
        log.Fatalf("Failed to setup cgroups: %v", err)
    }
    ```
    *Detail*: (See Section 2.6 for more on Cgroups).

3.  **Internal Filesystem Setup (`setupRootFS()`):**
    Next, the child calls `setupRootFS()`. This is crucial for isolating the container's view of the filesystem from the host. It uses the `pivot_root` system call to switch the root directory of the container.
    ```go
    // my-runc/main.go (within case "child")
    if err := setupRootFS("/"); err != nil {
        log.Fatalf("Failed to setup root filesystem: %v", err)
    }
    ```
    *Detail*: (See Section 2.5 for more on `pivot_root`).

4.  **Waiting for Parent's External Setup to Complete:**
    After completing its internal setup (cgroups and rootfs), the child process then actively waits for a signal from the parent. It reads from the inherited pipe (file descriptor 3), which causes it to block until the parent writes to it. This ensures that any external configuration (like network setup by the parent) is finished before the user's command starts.
    ```go
    // my-runc/main.go (within case "child")
    pipe := os.NewFile(3, "pipe") // Child opens inherited FD 3
    log.Println("Waiting for network setup signal from parent...")
    if _, err := io.ReadAll(pipe); err != nil { // This read blocks until parent writes
        log.Fatalf("Child failed to read from pipe during network wait: %v", err)
    }
    pipe.Close() // Close the pipe after receiving the signal
    log.Println("Network setup signal received. Proceeding to execute command.")
    ```
    *Detail*: This is the critical synchronization point discussed in Section 2.2.

5.  **Executing the User's Command (Second `exec`):**
    Once unblocked by the parent's signal, the child process is now ready to run the user's actual command. It achieves this by calling `exec.Command(...).Run()`.
    ```go
    // my-runc/main.go (within case "child")
    cmd := exec.Command(commandToRun[0], commandToRun[1:]...) // Prepare user's command
    cmd.Stdin = os.Stdin // Inherit stdin, stdout, stderr
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil { // This replaces the current child my-runc process
        log.Fatalf("Failed to run command in container: %v", err)
    }
    ```
    *Detail*: This is the **second and final `exec` call** in the container's lifecycle. The current "child" `my-runc` process is *completely replaced* by the user's command (e.g., `bash`, `ping`, etc.). The `my-runc` binary code is no longer running at this point; only the user's desired program is executing within the fully isolated container environment.

### 2.4 Linux Namespaces: The Walls of Isolation

Linux Namespaces are the fundamental primitives that provide the "isolation" in containers. They virtualize system resources, making a process believe it has its own dedicated copy of a resource, when in reality it's sharing the underlying kernel.

These namespaces are established by the **Parent Process** when it configures the `syscall.SysProcAttr` for the child process.

**File:** `my-runc/run_linux.go` (configuring `SysProcAttr`)

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

*   **`CLONE_NEWPID` (Process ID Namespace)**:
    *   **Effect:** Provides the container with its own isolated process ID tree. The first process in the new namespace (the user's command) will see itself as PID 1.
    *   **Mechanism:** The kernel maintains a translation table. PID 1 inside the container might be PID 4502 on the host. Processes in one PID namespace cannot see or interact with processes in another, except through specific namespace-aware mechanisms.

*   **`CLONE_NEWNS` (Mount Namespace)**:
    *   **Effect:** Gives the container its own private set of filesystem mount points. Changes (like mounting or unmounting) within this namespace do not affect the host's filesystem view.
    *   **Mechanism:** This is fundamental for `pivot_root`, allowing the container to have its own root filesystem entirely separate from the host.

*   **`CLONE_NEWUTS` (UTS Namespace)**:
    *   **Effect:** Isolates the hostname and NIS domain name. The container can have its own hostname, distinct from the host's.

*   **`CLONE_NEWIPC` (IPC Namespace)**:
    *   **Effect:** Isolates System V Inter-Process Communication (IPC) objects (message queues, semaphores, shared memory) and POSIX message queues. Processes in different IPC namespaces cannot interfere with each other's IPC mechanisms.

*   **`CLONE_NEWNET` (Network Namespace)**:
    *   **Effect:** Provides the container with its own isolated network stack, including network interfaces, IP addresses, routing tables, and firewall rules. It starts as a completely empty network environment.
    *   **Mechanism:** The parent process (`my-runc run`) is responsible for configuring network devices (like `veth` pairs) *into* this newly created, empty network namespace.

*   **`CLONE_NEWUSER` (User Namespace)**:
    *   **Effect:** Isolates user and group IDs. This is a crucial security feature. It allows a process to have UID 0 (root) privileges *inside* the container, while mapping to an unprivileged user ID on the host system.
    *   **Mechanism:** The `UidMappings` and `GidMappings` within `SysProcAttr` explicitly define this translation. For example, container UID 0 might map to host UID 1000. This means a malicious actor gaining root inside the container does not gain root on the host.

### 2.5 Filesystem Isolation: `pivot_root` Internals

**File:** `my-runc/namespace_linux.go` -> `setupRootFS()`

This function, executed by the **Child Process**, is responsible for establishing the container's isolated root filesystem. It uses `pivot_root`, which provides stronger isolation than `chroot` because it completely detaches the old root filesystem.

**The `setupRootFS()` Flow:**

```go
// my-runc/namespace_linux.go
func setupRootFS(rootfs string) error {
	log.Println("Setting up root filesystem...")

	// 1. Prepare New Root Directory:
    // A temporary directory (`/tmp/my-runc-root`) is created. This will momentarily serve as the
    // mount point for the container's desired root filesystem before `pivot_root` makes it the actual `/`.
	newRoot := "/tmp/my-runc-root"
	if err := os.MkdirAll(newRoot, 0755); err != nil { /* ... */ }

	// 2. Bind Mount the Container Image:
    // The path to the container's root filesystem image (`rootfs`, e.g., a directory containing busybox)
    // is bind-mounted onto `newRoot`. This means the contents of `rootfs` now appear at `newRoot`.
    // `syscall.MS_BIND` creates the bind mount; `syscall.MS_REC` makes it recursive for submounts.
	if err := syscall.Mount(rootfs, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil { /* ... */ }

	// 3. Make New Root's Mount Point Private:
    // This ensures that any mount/unmount operations within this `newRoot` do not
    // propagate to other mount namespaces (especially the host's mount namespace).
	if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil { /* ... */ }

	// 4. Create Directory for Old Root:
    // A temporary directory (`.pivot_root`) is created *inside* the `newRoot`. This is a requirement
    // for `pivot_root`; the old root filesystem will be moved and mounted here.
	putOld := filepath.Join(newRoot, ".pivot_root")
	if err := os.Mkdir(putOld, 0777); err != nil && !os.IsExist(err) { /* ... */ }

	// 5. Perform `pivot_root` System Call:
    // This is the atomic swap. `newRoot` becomes the new `/` for the process,
    // and the original root filesystem is moved and mounted at `putOld` (`/.pivot_root` relative to the new root).
	if err := syscall.PivotRoot(newRoot, putOld); err != nil { /* ... */ }

	// 6. Change Current Working Directory:
    // The current working directory is set to `/` (the new root) to ensure consistency.
	if err := os.Chdir("/"); err != nil { /* ... */ }

	// 7. Unmount the Old Root:
    // The old root (now accessible at `/.pivot_root`) is unmounted. `syscall.MNT_DETACH` allows
    // unmounting even if it's "busy", effectively severing the link to the host's original filesystem.
	if err := syscall.Unmount("/.pivot_root", syscall.MNT_DETACH); err != nil { /* ... */ }

	// 8. Remove Temporary Directory:
    // The empty `.pivot_root` directory is removed.
	if err := os.Remove("/.pivot_root"); err != nil { /* ... */ }

	// 9. Mount `/proc` Filesystem:
    // A new `/proc` filesystem is mounted. This is crucial for processes within the container
    // to correctly see their own process information (PIDs, etc.) and interact with the kernel,
    // reflecting the container's isolated PID namespace.
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil { /* ... */ }

	log.Println("Root filesystem setup complete")
	return nil
}
```
*   **Result:** The container now has a completely independent root filesystem. Any attempt by a process inside the container to `cd ../../../` will remain within this new isolated filesystem; it is physically impossible to access the host's original file hierarchy.

### 2.6 Control Groups (Cgroups): Resource Metering

**File:** `my-runc/namespace_linux.go` -> `setupCgroups()`

Namespaces are about **Isolation**, preventing processes from seeing each other's resources. Cgroups (Control Groups) are about **Resource Limits**, controlling how much of a resource processes can *use*. This function, executed by the **Child Process**, configures resource constraints for the container.

**The `setupCgroups()` Flow (Interacting with the Kernel via Filesystem):**

The Linux kernel exposes Cgroups as a virtual filesystem, typically mounted at `/sys/fs/cgroup`. `setupCgroups()` interacts with this filesystem by creating directories and writing values to special control files.

```go
// my-runc/namespace_linux.go
func setupCgroups() error {
	log.Println("Setting up cgroups...")

	cgroupPath := "/sys/fs/cgroup"
	var memPath string
	var limitFile string
	
	// 1. Determine Cgroups Version (v1 or v2):
    // The code checks for the existence of `/sys/fs/cgroup/memory` to determine
    // if Cgroups v1 (with a separate memory controller hierarchy) or v2 (unified hierarchy) is in use.
	if _, err := os.Stat(filepath.Join(cgroupPath, "memory")); err == nil {
		log.Println("Detected Cgroups v1")
		memPath = filepath.Join(cgroupPath, "memory", "my-container") // Path for v1 memory cgroup
		limitFile = "memory.limit_in_bytes" // Cgroups v1 memory limit file
	} else {
		log.Println("Detected Cgroups v2")
		memPath = filepath.Join(cgroupPath, "my-container") // Path for v2 unified cgroup
		limitFile = "memory.max" // Cgroups v2 memory limit file
	}

	// 2. Create the Container's Cgroup Directory:
    // A new directory (e.g., `/sys/fs/cgroup/memory/my-container` for v1, or `/sys/fs/cgroup/my-container` for v2)
    // is created. The kernel automatically populates this directory with control files relevant to the cgroup.
	if err := os.Mkdir(memPath, 0755); err != nil && !os.IsExist(err) { /* ... */ }

	// 3. Assign Current Process to Cgroup:
    // The PID of the current process (the child `my-runc` process) is written into
    // the `cgroup.procs` (for v1) or `cgroup.threads` (for v2 if `cgroup.procs` is absent) file within the new cgroup.
    // This associates the process and all its future descendants (including the user's command)
    // with this cgroup, subjecting them to its limits.
	pid := os.Getpid()
	procsFile := filepath.Join(memPath, "cgroup.procs")
	if _, err := os.Stat(procsFile); os.IsNotExist(err) {
		procsFile = filepath.Join(memPath, "cgroup.threads")
	}
	if err := os.WriteFile(procsFile, []byte(fmt.Sprintf("%d", pid)), 0700); err != nil { /* ... */ }

	// 4. Set Memory Limit:
    // A memory limit (e.g., 100MB) is written to the appropriate memory limit control file
    // (`memory.limit_in_bytes` for v1 or `memory.max` for v2).
	memoryLimitBytes := "100000000" // 100MB
	limitFilePath := filepath.Join(memPath, limitFile)
	if err := os.WriteFile(limitFilePath, []byte(memoryLimitBytes), 0700); err != nil { /* ... */ }

	log.Println("Cgroups setup complete")
	return nil
}
```
*   **The Enforcement:** Once configured, the Linux kernel actively monitors the processes within this cgroup. If a process attempts to allocate memory beyond the set limit, the kernel will trigger the **OOM Killer** (Out-Of-Memory Killer) and send a `SIGKILL` signal to terminate the offending process, thus enforcing the resource constraint.

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

## Chapter 5: Infrastructure & Testing (The Engineering Backbone)

As we moved from a toy project to a functional system, we introduced robust engineering practices.

### 5.1 The Build System: Makefiles & Cross-Compilation

We replaced manual `go build` commands with a recursive **Makefile** system. 
*   **Root Makefile:** Orchestrates the entire project (`make all`, `make test`).
*   **Component Makefiles:** Each component (`my-runc`, `my-kube`) has its own Makefile responsible for building its specific binaries into the shared `bin/` directory.

**Cross-Compilation Challenge:**
When developing on macOS (`darwin/arm64`) but targeting Linux VMs (`linux/arm64`), we utilize Go's built-in cross-compilation support by exporting environment variables in our build commands:
`GOOS=linux GOARCH=arm64 make all`

### 5.2 End-to-End Testing: Mocking the Runtime

To test `my-kube`'s orchestration logic without needing a Linux kernel (e.g., on a Mac CI/CD runner), we implemented an **End-to-End (E2E) Test Suite**.

*   **The Mock Runtime (`mock-runc.sh`):** Instead of calling the real `my-runc` (which requires `clone` and `namespaces`), we inject a shell script that mimics the runtime's interface (`run <cmd>`). This script logs its invocations to a file.
*   **Integration Test (`e2e_test.go`):**
    1.  Compiles the actual `my-kube-server` and `my-kubelet`.
    2.  Spins them up on localhost ports.
    3.  Submits a Pod via the API.
    4.  Verifies the "Mock Runtime" log file to confirm the Kubelet received the instruction and "executed" the container.

This decouples the **logic of orchestration** from the **mechanics of containerization**.

### 5.3 Virtualization Architecture: The "Split-Brain" Network Problem

Running a cluster on macOS using Lima VMs introduced a classic distributed systems problem: **Network Isolation**.

**The Problem:**
By default, Lima VMs using `vz` (Virtualization.framework) or `qemu` (user-mode networking) sit behind a NAT. They can reach the internet, but **they cannot reach each other**.
*   Master Node thinks its IP is `192.168.5.15`.
*   Worker Node thinks *its* IP is also `192.168.5.15`.
*   They are in parallel, isolated network namespaces provided by the host.

**The Solution: Shared Bridged Networking (`socket_vmnet`)**
To create a true cluster, we utilized `socket_vmnet` to create a shared bridge on the host. This puts all VMs on the same virtual subnet (`192.168.105.x`), allowing:
1.  **Unique IPs:** Master gets `.2`, Worker gets `.3`.
2.  **Inter-Node Communication:** Workers can dial the Master's API.
3.  **Host-to-VM Communication:** We can curl the Master's API from the host.

---

## Chapter 6: The Debugging Chronicles (Post-Mortems)

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
