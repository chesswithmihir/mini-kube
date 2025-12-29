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

Our `main()` function uses a `switch` statement to handle the "Parent" vs. "Child" logic.

```go
switch command {
case "run":
    // The Parent: Starts the process
    run(commandToRun, ip)
case "child":
    // The Child: Sets up the room from the inside
    setupCgroups()
    setupRootFS("/")
    // ... exec user command
}
```

### 2.2 The "Re-Execution" Pattern (The Fork/Exec Loop)

**File:** `my-runc/run_linux.go`

This is the "Loop" that confuses everyone. To create a container, the program runs itself again.

1.  **Parent:** Executes `exec.Command("/proc/self/exe", "child", "bash")`.
2.  **The Mystery of `/proc/self/exe`**: This is a symlink that always points to the current binary. So `my-runc` is spawning a copy of `my-runc`.
3.  **Why?**: We need to run Go code *inside* the new namespaces (PID, Net, etc.) to perform setup (like mounting `/proc`) before we finally turn into `bash`.

### 2.3 Linux Namespaces: The Six Walls of Isolation

**File:** `my-runc/run_linux.go`

In the `run()` function, we configure the `SysProcAttr`. This tells the Linux Kernel: "When you create this child, give it these private rooms."

```go
Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | ...
```

*   **`CLONE_NEWPID` (The Matrix)**: 
    *   **Metaphor:** Every process gets a new ID starting at 1.
    *   **Depth:** The Kernel maintains a translation table. Inside, you are PID 1. Outside, you are PID 4502. If you try to kill PID 2 (the host's second process), the Kernel says "PID 2? Never heard of her."
*   **`CLONE_NEWNET` (The Silence)**: 
    *   **Metaphor:** Cutting the phone lines.
    *   **Depth:** The process has no network devices. It cannot even talk to `localhost` until we manually bring the `lo` interface up.
*   **`CLONE_NEWUSER` (The Fake Identity)**:
    *   **Metaphor:** You are a King in your room, but a Peasant in the hallway.
    *   **Depth:** Maps Container-UID 0 (Root) to Host-UID 1000 (You). You can run `apt install` inside because you are "root", but you cannot delete `/etc/shadow` on the host because the host sees you as a regular user.

### 2.4 Filesystem Isolation: `pivot_root` internals

**File:** `my-runc/namespace_linux.go` -> `setupRootFS()`

We don't use `chroot` because it is "leaky". We use `pivot_root`. 

**The Step-by-Step Logic:**
1.  **Mount the rootfs:** We take a folder (like `/tmp/rootfs`) and "bind mount" it to itself. This makes the Kernel treat it as a formal disk mount point.
2.  **`syscall.PivotRoot`**: This is the atomic swap. 
    *   Old Root (`/`) moves to `/.pivot_root`.
    *   New Root (`/tmp/rootfs`) becomes the new `/`.
3.  **Severing the Link**: We call `syscall.Unmount("/.pivot_root", syscall.MNT_DETACH)`. 
    *   The old host filesystem is now **gone**. If the process tries to `cd ../../../`, it just hits the new `/`. It is physically impossible to see the host's files.

### 2.5 Control Groups (Cgroups): Resource Metering

**File:** `my-runc/namespace_linux.go` -> `setupCgroups()`

Namespaces are about **Isolation**. Cgroups are about **Limits**.

**How we talk to the Kernel:**
The Kernel exposes Cgroups as a filesystem at `/sys/fs/cgroup`.
1.  We `mkdir` a folder there: `/sys/fs/cgroup/memory/my-container`.
2.  The Kernel sees the new folder and instantly populates it with control files.
3.  We write the process's PID into `cgroup.procs`.
4.  We write `100000000` (100MB) into `memory.limit_in_bytes`.
5.  **The Enforcement**: The Kernel now watches every byte of RAM that process allocates. The moment it hits 100MB + 1 byte, the Kernel triggers the **OOM Killer** and sends a `SIGKILL` to the process.

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
