# Mini-Kube Architecture & Internals: A Textbook

**Author:** Gemini  
**Date:** December 28, 2025  
**Target Audience:** Computer Science Engineers & Systems Programmers

---

## Table of Contents

1.  [Introduction](#1-introduction)
2.  [Part I: The Kernel & The Runtime (`my-runc`)](#part-i-the-kernel--the-runtime-my-runc)
    *   [The Container Illusion](#the-container-illusion)
    *   [The Re-Execution Pattern](#the-re-execution-pattern-solving-the-chicken-and-egg)
    *   [System Calls & Namespaces (The "Matrix")](#system-calls--namespaces-the-matrix)
    *   [Filesystem Isolation: `pivot_root` vs `chroot`](#filesystem-isolation-pivot_root-vs-chroot)
    *   [Control Groups (Cgroups): The Resource Police](#control-groups-cgroups-the-resource-police)
    *   [Networking: Building a Virtual ISP](#networking-building-a-virtual-isp)
        *   [Case Study: The NAT & Subnet Debugging Saga](#case-study-the-nat--subnet-debugging-saga)
3.  [Part II: The Orchestrator (`my-kube`)](#part-ii-the-orchestrator-my-kube)
    *   [Distributed Systems Theory: Control Plane & Agents](#distributed-systems-theory-control-plane--agents)
    *   [The API Server: State Management & Concurrency](#the-api-server-state-management--concurrency)
    *   [The Scheduler: Algorithms & Assignment](#the-scheduler-algorithms--assignment)
    *   [The Kubelet Agent: The Sync Loop Pattern](#the-kubelet-agent-the-sync-loop-pattern)
4.  [Part III: End-to-End Request Tracing](#part-iii-end-to-end-request-tracing)

---

## 1. Introduction

`mini-kube` is an educational implementation of a Container Orchestrator (like Kubernetes) and a Container Runtime (like Docker/Runc). It is built from scratch in Go to demonstrate the low-level operating system primitives that power modern cloud infrastructure.

This document dissects the source code line-by-line, explaining the *why* behind every design decision, algorithm, and system call.

---

## Part I: The Kernel & The Runtime (`my-runc`)

The `my-runc` directory contains the low-level machinery. Its job is to take a command (like `bash`) and run it in an isolated environment.

### The Container Illusion

There is no such thing as a "Linux Container" in the kernel. There are only **Processes** with restricted views. A container is simply a process that has been lied to about:
1.  **Who it is** (PID Namespace)
2.  **What it can see** (Mount Namespace)
3.  **Who its neighbors are** (Network Namespace)
4.  **What resources it owns** (User Namespace)

### The Re-Execution Pattern: Solving the Chicken and Egg

**File:** `my-runc/main.go`, `my-runc/run_linux.go`

One of the most confusing parts of reading container runtime code is seeing the program call *itself*.

```go
// my-runc/run_linux.go
cmd := exec.Command("/proc/self/exe", append([]string{"child"}, commandToRun...)...)
```

**Why do we do this?**
In Linux, you cannot change the PID Namespace of the *currently running* process. It is an immutable property of your birth. If you want a process to be "PID 1" in a new world, you must `fork/clone` a **child** into that new world.

However, we can't just run the user's command (e.g., `bash`) immediately as that child. Why?
1.  We haven't set up the filesystem (Process will see host files).
2.  We haven't set up Cgroups (Process can eat all RAM).
3.  We haven't set up Networking.

**The Solution:**
1.  **Stage 1 (Parent):** `my-runc run ...`
    *   Talks to the Kernel.
    *   Prepares the namespaces flags.
    *   Clones a child.
2.  **Stage 2 (Child - Intermediate):** `my-runc child ...`
    *   Born inside the namespaces.
    *   Still running Go code (smart).
    *   Sets up the environment (Mounts, Cgroups).
    *   Waits for Network handshake.
3.  **Stage 3 (Child - Final):** `exec("bash")`
    *   Replaces the Go memory image with the user's program.
    *   The user's program wakes up inside a fully prepared box.

### System Calls & Namespaces (The "Matrix")

**File:** `my-runc/run_linux.go`

The most critical lines of code in the entire project are in the `SysProcAttr` struct. This is the bridge between Go and the Linux `clone()` syscall.

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWPID |  // Isolation 1
                syscall.CLONE_NEWNS  |  // Isolation 2
                syscall.CLONE_NEWUTS |  // Isolation 3
                syscall.CLONE_NEWIPC |  // Isolation 4
                syscall.CLONE_NEWNET |  // Isolation 5
                syscall.CLONE_NEWUSER,  // Isolation 6
    
    // User Namespace Mapping
    UidMappings: []syscall.SysProcIDMap{
        {ContainerID: 0, HostID: os.Getuid(), Size: 1},
    },
}
```

#### Detailed Breakdown of Flags:

1.  **`CLONE_NEWPID`**:
    *   **Effect:** The child process becomes **PID 1** inside the new namespace.
    *   **Implication:** PID 1 is special. It is the "init" process. If PID 1 dies, the whole namespace dies. It is also responsible for reaping zombie processes.
    *   **Without this:** The process would see its real host PID (e.g., 14002) and could see/kill other host processes.

2.  **`CLONE_NEWNS` (Mount Namespace)**:
    *   **Effect:** The process gets its own copy of the list of mounted filesystems.
    *   **Implication:** If the process runs `mount` or `unmount`, it only affects *its* view. This is required for `pivot_root` to work without unmounting the user's actual hard drive.

3.  **`CLONE_NEWNET` (Network Namespace)**:
    *   **Effect:** The process gets an empty network stack. No `eth0`, no `wlan0`, just a broken `lo`.
    *   **Implication:** Total isolation. It cannot access the internet until we physically inject a virtual wire (veth pair) into this namespace.

4.  **`CLONE_NEWUSER` (User Namespace)**:
    *   **Effect:** Decouples User IDs.
    *   **The Mapping:** `ContainerID: 0` maps to `HostID: 1000` (You).
    *   **Security:** Inside the container, the process thinks it is `root` (UID 0). It can modify routing tables, mount filesystems, etc. *But only inside its sandbox*. If it manages to break out to the host, it is instantly treated as UID 1000 (standard user) and blocked from doing damage.

---

### Filesystem Isolation: `pivot_root` vs `chroot`

**File:** `my-runc/namespace_linux.go` -> `setupRootFS()`

We use `syscall.PivotRoot`, not `chroot`.

*   **`chroot`**: "Put on blinders." The process technically still stands on the host filesystem, but the kernel checks every path to ensure it starts with `/new/root`. It is insecure; it is possible to break out of a chroot jail.
*   **`pivot_root`**: "Swap the floor." The kernel unmounts the host filesystem for this process and mounts the new filesystem in its place. The old filesystem is literally gone from the namespace.

**The Algorithm:**
1.  **Bind Mount New Root:** `mount --bind /tmp/new-root /tmp/new-root`. (Strict kernel requirement: `pivot_root` only works on mount points, not directories).
2.  **Pivot:** `syscall.PivotRoot("/tmp/new-root", "/tmp/new-root/.pivot_root")`.
    *   This atomic swap moves the current root to `.pivot_root` and makes `/tmp/new-root` the new `/`.
3.  **Unmount Old:** `syscall.Unmount("/.pivot_root", MNT_DETACH)`.
    *   Sever the link to the host.
4.  **Mount /proc:** `mount -t proc proc /proc`.
    *   Crucial! Linux tools like `ps`, `top`, and `free` read from `/proc`. If we don't mount a fresh proc filesystem, `ps` will either fail or show the host's process table (breaking the illusion).

---

### Control Groups (Cgroups): The Resource Police

**File:** `my-runc/namespace_linux.go` -> `setupCgroups()`

Namespaces limit what you **see**. Cgroups limit what you **use**.

**The Filesystem Interface:**
The kernel exposes cgroups as a file system at `/sys/fs/cgroup`. We don't need a library; we just write to files.

**V1 vs V2:**
*   **Legacy (V1):** Resources are split (`/sys/fs/cgroup/memory`, `/sys/fs/cgroup/cpu`).
*   **Modern (V2):** Unified hierarchy (`/sys/fs/cgroup`).

**Our Implementation:**
1.  **Detect:** Check if `/sys/fs/cgroup/memory` exists.
2.  **Create Group:** `mkdir /sys/fs/cgroup/my-container`.
3.  **Assign Process:** Write `os.Getpid()` to `/sys/fs/cgroup/my-container/cgroup.procs`.
4.  **Limit Memory:** Write `100000000` (100MB) to `memory.max` (V2) or `memory.limit_in_bytes` (V1).

If the process exceeds 100MB, the kernel's **OOM Killer** (Out of Memory Killer) will target this specific cgroup and terminate the process.

---

### Networking: Building a Virtual ISP

**File:** `my-runc/network_linux.go`

This is the most complex part of the runtime. We manually constructed a TCP/IP network stack.

#### The Physical Analogy
*   **The Bridge (`my-bridge0`):** A physical Network Switch plugged into the wall (Host).
*   **The Veth Pair:** An Ethernet cable with two ends.
*   **The Namespace:** A locked room where the computer (Container) sits.

#### The Algorithm (`setupNetwork`)
1.  **Create Bridge:** `ip link add my-bridge0 type bridge`.
2.  **Create Cable:** `ip link add veth1234 type veth peer name veth-c1234`.
    *   Now we have a cable lying on the floor of the host.
3.  **Plug into Switch:** `ip link set veth1234 master my-bridge0`.
4.  **Throw Cable into Room:** `ip link set veth-c1234 netns <PID>`.
    *   This is the magic. One end of the virtual cable disappears from the host and appears inside the container's isolated network namespace.
5.  **Rename & Configure:**
    *   Inside the container (via `nsenter`): Rename `veth-c1234` to `eth0`.
    *   Assign IP: `10.244.0.100/16`.
    *   Route: `ip route add default via 10.244.0.1` (The Bridge IP).

#### Case Study: The NAT & Subnet Debugging Saga

During development, we encountered two critical networking failures.

**Failure 1: The Missing Subnet Mask**
*   **Symptom:** The container (`10.244.0.100`) could not ping the gateway (`10.244.0.1`). Error: `Nexthop has invalid gateway`.
*   **Root Cause:** We assigned the IP as `10.244.0.100` (implied `/32`).
    *   In IP routing, `/32` means "This IP is the only thing in the network."
    *   The container looked at its routing table: "I need to reach `10.244.0.1`. My mask is `/32`. That IP is outside my network. I have no way to reach it."
*   **Fix:** We appended `/16` (`10.244.0.100/16`).
    *   Now the container says: "My network covers `10.244.0.0` to `10.244.255.255`. The gateway `10.244.0.1` is my neighbor. I can talk to it directly via ARP."

**Failure 2: The Packet Loss (No NAT)**
*   **Symptom:** Container could ping Gateway, but `ping 8.8.8.8` failed with 100% packet loss.
*   **Root Cause:**
    1.  Container sends packet: `Src: 10.244.0.100` -> `Dst: 8.8.8.8`.
    2.  Host forwards packet to Internet.
    3.  Google receives packet.
    4.  Google tries to reply to `10.244.0.100`.
    5.  **FAILURE:** `10.x.x.x` is a private, non-routable IP address. Core internet routers drop packets destined for private ranges. Google had no idea who sent it.
*   **Fix:** **IP Masquerading (NAT)**.
    *   Command: `iptables -t nat -A POSTROUTING -s 10.244.0.0/16 -j MASQUERADE`.
    *   **Logic:** As the packet leaves the Host VM, the kernel **rewrites the Source IP** from `10.244.0.100` to the Host's Public IP (e.g., `192.168.5.2`).
    *   Google replies to `192.168.5.2`.
    *   The Host Kernel remembers the connection, rewrites the destination back to `10.244.0.100`, and forwards it to the container.

---

## Part II: The Orchestrator (`my-kube`)

`my-kube` operates at a higher abstraction level. It doesn't care about syscalls; it cares about **State**.

### Distributed Systems Theory: Control Plane & Agents

We use a **Hub-and-Spoke** architecture.
*   **Hub (Server):** The source of truth. It stores the "Desired State" (What the user wants).
*   **Spoke (Agent):** The reconciler. It compares "Desired State" vs "Actual State" and takes action.

### The API Server: State Management & Concurrency

**File:** `my-kube/pkg/server/store.go`

We implemented an in-memory Key-Value store.

**Concurrency Control:**
Since Go's HTTP server handles each request in a separate Goroutine, multiple agents could try to register at the exact same time.
*   **Problem:** Race conditions writing to the `map`.
*   **Solution:** `sync.RWMutex`.
    *   `Lock()`: Exclusive lock for writing (Adding a Pod). No one else can read or write.
    *   `RLock()`: Shared lock for reading (Listing Pods). Multiple readers allowed, but no writers.

**File:** `my-kube/pkg/server/handler.go`

We built a REST API.
*   `POST /pods`: Submit a job.
*   `GET /nodes/{id}/pods`: The "Inbox" for a worker node.

**Design Decision: Polling vs Pushing**
*   We chose **Polling** (Agent asks Server) instead of Pushing (Server calls Agent).
*   **Why?**
    *   If Agent is behind a firewall (NAT), Server can't connect to it.
    *   If Server goes down, Agent keeps running (decoupled).
    *   This is exactly how Kubernetes works.

### The Scheduler: Algorithms & Assignment

**File:** `my-kube/server/main.go` -> `runScheduler()`

The Scheduler is an infinite loop running in the background of the Server.

**The Algorithm:**
1.  Acquire Lock (Read State).
2.  Find all Pods where `Status == Pending` AND `NodeID == ""`.
3.  Find all Nodes.
4.  **Decision Logic:**
    *   Current Implementation: **Round Robin / First Available**. Pick `nodes[0]`.
    *   Real Kubernetes: Complex scoring (CPU capacity, Taints/Tolerations, Affinity).
5.  **Bind:** Update the Pod's `NodeID` in the store.

### The Kubelet Agent: The Sync Loop Pattern

**File:** `my-kube/agent/agent.go`

The Agent is a state machine. It does not just "receive commands". It **Synchronizes**.

**The Loop:**
1.  **Get Desired State:** `GET /nodes/{my-id}/pods`. Server says: "You should be running [Pod A, Pod B]".
2.  **Get Actual State:** Agent checks internal map `knownPods`. "I am running [Pod A]".
3.  **Diff:** "I am missing Pod B."
4.  **Reconcile:** Call `runtime.RunPod(Pod B)`.
5.  **Update Cache:** Add Pod B to `knownPods`.

This loop runs every 5 seconds. If a pod crashes (Actual State changes), the next loop will see it's missing (or stopped) and restart it. This creates a **Self-Healing System**.

---

## Part III: End-to-End Request Tracing

Let's trace the life of a request: `curl -X POST -d '{"command": ["python", "server.py"]}' localhost:8080/pods`

1.  **User** hits API Server.
2.  **API Server** decodes JSON, acquires `Mutex.Lock()`, writes Pod to `MemoryStore` with Status `Pending`.
3.  **Scheduler Loop** wakes up. Sees Pending Pod. Sees Worker Node `worker-1`.
4.  **Scheduler** updates Pod in Store: `NodeID = "worker-1"`.
5.  **Agent (worker-1)** wakes up (5s poll). Calls `GET /nodes/worker-1/pods`.
6.  **Agent** receives JSON containing the Pod.
7.  **Agent** calls `Runtime.RunPod()`.
8.  **Runtime (`my-runc`)** is executed via `os/exec`.
9.  **`my-runc` (Parent)** calls `clone(CLONE_NEWPID | ...)` to create Child.
10. **`my-runc` (Parent)** sets up Bridge and Veth pair (`setupNetwork`).
11. **`my-runc` (Child)** sets up Cgroups (Limit RAM) and Pivot Root (Secure Filesystem).
12. **`my-runc` (Child)** calls `exec("python server.py")`.
13. **Kernel** starts Python process as PID 1 inside the isolated namespace.

The system is now live.
