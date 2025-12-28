# Mini-Kube Internals: A Deep Dive Textbook

**Author:** Gemini  
**Date:** December 28, 2025  
**Version:** 2.0 (Textbook Edition)

---

## Preface

This document is written for computer science engineers, systems programmers, and anyone who wants to understand the "magic" of Kubernetes and Containers by reading the source code of a simplified implementation. 

We will traverse the stack from the **Linux Kernel System Calls** up to the **Distributed Control Loops** of the orchestrator. We will not just explain *what* the code does, but *why* it was written that way, and what operating system principles it leverages.

---

## Table of Contents

1.  [Chapter 1: The Container Runtime (`my-runc`)](#chapter-1-the-container-runtime-my-runc)
    *   [1.1 The Architecture of `main.go`](#11-the-architecture-of-maingo)
    *   [1.2 The "Re-Execution" Pattern (Fork/Exec/Clone)](#12-the-re-execution-pattern-forkexecclone)
    *   [1.3 Linux Namespaces: The Isolation Primitives](#13-linux-namespaces-the-isolation-primitives)
    *   [1.4 The Filesystem: `pivot_root` Deep Dive](#14-the-filesystem-pivot_root-deep-dive)
    *   [1.5 Control Groups (Cgroups): Resource Management](#15-control-groups-cgroups-resource-management)
    *   [1.6 Networking: Building the Virtual ISP](#16-networking-building-the-virtual-isp)
        *   [1.6.1 L2 Switching (The Bridge)](#161-l2-switching-the-bridge)
        *   [1.6.2 Virtual Cabling (Veth Pairs)](#162-virtual-cabling-veth-pairs)
        *   [1.6.3 Network Address Translation (NAT) & The Packet Flow](#163-network-address-translation-nat--the-packet-flow)
2.  [Chapter 2: The Orchestrator (`my-kube`)](#chapter-2-the-orchestrator-my-kube)
    *   [2.1 Distributed System Design: Hub-and-Spoke](#21-distributed-system-design-hub-and-spoke)
    *   [2.2 The API Server: Concurrency & State](#22-the-api-server-concurrency--state)
    *   [2.3 The Scheduler: The Assignment Loop](#23-the-scheduler-the-assignment-loop)
    *   [2.4 The Kubelet Agent: Edge-Triggered Reconciliation](#24-the-kubelet-agent-edge-triggered-reconciliation)
3.  [Chapter 3: Critical Debugging Case Studies](#chapter-3-critical-debugging-case-studies)
    *   [3.1 The "Invalid Gateway" Subnet Error](#31-the-invalid-gateway-subnet-error)
    *   [3.2 The "Ping 8.8.8.8" Failure (NAT)](#32-the-ping-8888-failure-nat)

---

## Chapter 1: The Container Runtime (`my-runc`)

The `my-runc` binary is a userspace tool that interacts with the Linux Kernel to create isolated processes. It corresponds to the **OCI (Open Container Initiative)** runtime specification (like `runc`, `crun`).

### 1.1 The Architecture of `main.go`

The entry point (`my-runc/main.go`) acts as a CLI dispatcher. It handles two distinct phases of a container's life cycle that are often confused.

1.  **`my-runc run` (Phase 1: The Parent)**
    *   **Context:** Runs in the **Host's** namespaces.
    *   **Privileges:** Root (via `sudo`).
    *   **Responsibility:** Talk to the kernel to spawn the child. It does *not* execute the user's command (e.g., `bash`) directly. It sets up the sandbox boundaries.
    *   **Key Code:** Calls `run()` in `run_linux.go`.

2.  **`my-runc child` (Phase 2: The Setup Agent)**
    *   **Context:** Runs **Inside** the new, empty namespaces.
    *   **Privileges:** Root (Cap-Admin) *inside* the container context.
    *   **Responsibility:** "Furnish the room." The walls (namespaces) are up, but the room is empty. It mounts filesystems, sets up Cgroups, and configures the network interface *from the inside*.
    *   **Key Code:** Calls `setupCgroups()`, `setupRootFS()`, and finally `syscall.Exec()`.

### 1.2 The "Re-Execution" Pattern (Fork/Exec/Clone)

In `run_linux.go`, we see this line:
```go
cmd := exec.Command("/proc/self/exe", append([]string{"child"}, commandToRun...)...)
```

**Why do we execute `/proc/self/exe`?**
This is a pointer to the currently running binary (`my-runc`). The parent process is essentially telling the kernel: "I want you to start a new process. Use *my* binary code. But start it with different arguments (`child`)."

**The "Chicken and Egg" Problem of Namespaces:**
You cannot "enter" a PID namespace that doesn't exist yet. And you cannot change your own PID namespace once you are alive (you already have a PID).
*   **Solution:** You must specify the namespaces **at the moment of birth**.
*   This is why we set `SysProcAttr` on the `cmd` object before calling `cmd.Start()`.

### 1.3 Linux Namespaces: The Isolation Primitives

The Linux Kernel (`kernel/fork.c`) implements `clone()`. We access this via `syscall.SysProcAttr`.

```go
Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | ...
```

#### The Big Six Namespaces:

1.  **`CLONE_NEWPID` (Process IDs)**
    *   **Kernel View:** The kernel maintains a mapping. Host PID `14052` maps to Container PID `1`.
    *   **User View:** Running `ps` inside the container shows PID 1.
    *   **Why it matters:** Processes inside cannot signal (kill) processes outside because they literally cannot address them. PID `1` (init) also has special responsibilities (reaping zombies).

2.  **`CLONE_NEWNS` (Mounts)**
    *   **History:** The first namespace (hence just `NEWNS`).
    *   **Function:** Allows the process to have a private list of mount points.
    *   **Usage:** We use this to mount a new `/proc` and `pivot_root` to a new filesystem without affecting the host.

3.  **`CLONE_NEWNET` (Networking)**
    *   **Function:** Gives the process a completely empty network stack.
    *   **Initial State:** Only a `lo` (Loopback) interface exists, and it is DOWN.
    *   **Challenge:** The container is deaf and mute. We must "install a cable" (Veth Pair) later.

4.  **`CLONE_NEWUTS` (Unix Timesharing System)**
    *   **Function:** Allows changing the Hostname and Domain Name.
    *   **Usage:** `syscall.Sethostname("my-container")`.

5.  **`CLONE_NEWIPC` (Inter-Process Communication)**
    *   **Function:** Isolates Shared Memory segments and Message Queues.
    *   **Why:** Prevents a container from writing to the shared memory of another container (a common attack vector).

6.  **`CLONE_NEWUSER` (User IDs)**
    *   **Function:** Maps a user ID inside the container to a different one outside.
    *   **Code:**
        ```go
        UidMappings: []syscall.SysProcIDMap{
            {ContainerID: 0, HostID: os.Getuid(), Size: 1},
        }
        ```
    *   **Explanation:** "User 0 (Root) inside acts like User 1000 (Mihir) outside." This is a massive security feature. Even if the containerized process breaks out, it finds itself as a regular, non-root user on the host.

### 1.4 The Filesystem: `pivot_root` Deep Dive

Most tutorials use `chroot`. We use `pivot_root` because it is secure.

**The Vulnerability of `chroot`:**
`chroot` (Change Root) merely modifies the pathname lookup algorithm. If a process has a handle to a directory *outside* the jail, it can `chdir("..")` its way out.

**The `pivot_root` Mechanism (`namespace_linux.go`):**
This syscall swaps the mount namespace root.

1.  **Preparation:** `mkdir /tmp/my-runc-root`. Bind-mount the rootfs here.
2.  **The Pivot:** `syscall.PivotRoot("/tmp/my-runc-root", "/tmp/my-runc-root/.pivot_root")`.
    *   **Action:** The Kernel takes the *current* root (`/`) and moves it to `.pivot_root`.
    *   **Action:** The Kernel takes `/tmp/my-runc-root` and makes it the new `/`.
3.  **The Unmount:** `syscall.Unmount("/.pivot_root", MNT_DETACH)`.
    *   This is the mic drop. We unmount the old root. The container now has **no way** to reference the host filesystem. It has vanished.
4.  **Mounting `/proc`:**
    *   `syscall.Mount("proc", "/proc", "proc", ...)`
    *   This mounts the **virtual filesystem** that exposes kernel statistics. Crucially, because we are in a new PID Namespace, this `/proc` only shows information relevant to *this* namespace.

### 1.5 Control Groups (Cgroups): Resource Management

**File:** `namespace_linux.go`

While namespaces provide **Isolation** (Visibility), Cgroups provide **Accounting and Limits** (Usage).

**The Hierarchy:**
Cgroups are implemented as a **Virtual File System (VFS)**, typically mounted at `/sys/fs/cgroup`.

**Mechanism:**
1.  **Directory Creation:** `mkdir /sys/fs/cgroup/memory/my-container`.
    *   The kernel automatically populates this directory with control files (`memory.limit_in_bytes`, `cgroup.procs`, etc.).
2.  **Enrolling:** We write our own PID (`os.Getpid()`) into `cgroup.procs`.
    *   "Kernel, please count my resource usage in this bucket."
3.  **Limiting:** We write `100000000` (100MB) into `memory.limit_in_bytes`.
    *   "Kernel, if this bucket exceeds 100MB, kill the processes inside."

**V1 vs V2:**
Linux is transitioning to Cgroups V2 (Unified Hierarchy). Our code in `setupCgroups` detects the version:
*   **V1:** Separate hierarchies (`/sys/fs/cgroup/memory`, `/sys/fs/cgroup/cpu`).
*   **V2:** Single hierarchy (`/sys/fs/cgroup`). Memory limit file is named `memory.max`.

### 1.6 Networking: Building the Virtual ISP

This section explains `network_linux.go`. We built a Layer 2/3 network from scratch.

#### 1.6.1 L2 Switching (The Bridge)

We create a bridge named `my-bridge0`.
```bash
ip link add name my-bridge0 type bridge
ip addr add 10.244.0.1/16 dev my-bridge0
ip link set my-bridge0 up
```
*   **Concept:** A software bridge acts exactly like a physical network switch. It maintains a MAC address table and forwards Ethernet frames between ports.
*   **Gateway:** We assign `10.244.0.1` to the bridge itself. This becomes the "Default Gateway" for all containers.

#### 1.6.2 Virtual Cabling (Veth Pairs)

A **Veth Pair** is a pipe for network packets. What goes in one end comes out the other.
```bash
ip link add veth-host type veth peer name veth-container
```
1.  **Host End (`veth-host`):** We plug this into the bridge (`ip link set veth-host master my-bridge0`).
2.  **Container End (`veth-container`):** This starts on the host. We must **move** it into the container's namespace.
    ```go
    // network_linux.go
    exec.Command("ip", "link", "set", "veth-container", "netns", <Container_PID>)
    ```
    Once moved, the interface disappears from the host (`ip link show` won't see it) and appears inside the container.

#### 1.6.3 Network Address Translation (NAT) & The Packet Flow

When the container (`10.244.0.100`) pings Google (`8.8.8.8`), the packet flow is:

1.  **Routing Decision (Container):**
    *   Dest: `8.8.8.8`.
    *   Routing Table: `default via 10.244.0.1 dev eth0`.
    *   Action: Send to Gateway (Bridge).
2.  **Switching (Bridge):**
    *   Bridge receives frame. Passes it up to the Host Kernel IP stack.
3.  **Routing Decision (Host):**
    *   Dest: `8.8.8.8`.
    *   Routing Table: `default via 192.168.5.1 dev eth0` (The VM's gateway).
4.  **The NAT Problem:**
    *   The packet source is `10.244.0.100`. This is a private IP.
    *   If sent as-is, Google will try to reply to `10.244.0.100`. The internet routers will drop this.
5.  **The NAT Solution (IP Masquerade):**
    *   We added an `iptables` rule:
        ```bash
        iptables -t nat -A POSTROUTING -s 10.244.0.0/16 -j MASQUERADE
        ```
    *   **Action:** The Host Kernel replaces `Src: 10.244.0.100` with `Src: 192.168.5.2` (The VM's public IP). It saves this mapping in a conntrack table.
6.  **Return Trip:**
    *   Google replies to `192.168.5.2`.
    *   Host Kernel checks conntrack, sees the mapping, rewrites Dest to `10.244.0.100`, and forwards to the bridge.

---

## Chapter 2: The Orchestrator (`my-kube`)

`my-kube` is a simplified version of the Kubernetes Control Plane and Kubelet. It demonstrates **Declarative State Management**.

### 2.1 Distributed System Design: Hub-and-Spoke

*   **API Server (Hub):** Stateless (mostly), holds the "Desired State".
*   **Kubelet (Spoke/Agent):** Autonomous. Polls the Hub. Actively drives "Actual State" to match "Desired State".

### 2.2 The API Server: Concurrency & State

**File:** `my-kube/pkg/server/store.go`

We implemented an in-memory database to replace Etcd.

**The Concurrency Challenge:**
In Go, `http.ListenAndServe` spawns a goroutine for every request. If Agent A sends a Heartbeat and Agent B reports a status simultaneously, they might write to the `map` at the same time. Go maps are **not** thread-safe.

**The Solution: `sync.RWMutex`**
We use a **Read-Write Mutex**.
*   **`Lock()` (Write Lock):** Used when Adding/Updating Pods. Only one goroutine can hold this. All others block.
*   **`RLock()` (Read Lock):** Used when Listing Pods. Multiple goroutines can hold this simultaneously. Efficient for high-read workloads (like agents polling).

**REST API Design (`handler.go`):**
*   `POST /pods`: User submits a job.
*   `GET /nodes/{node_id}/pods`: The crucial endpoint. This is how the Server communicates with the Agent without dialing it directly. This allows Agents to be behind NATs or Firewalls.

### 2.3 The Scheduler: The Assignment Loop

**File:** `my-kube/server/main.go`

The scheduler is a background loop (a `goroutine`) separate from the HTTP handlers.

**Logic:**
1.  **Poll:** Every 5 seconds, scan the Store.
2.  **Filter:** Find Pods where `Status == Pending` AND `NodeID == ""`.
3.  **Select:** Find available Nodes.
4.  **Bind:** Assign `Pod.NodeID = Node.ID`.

**Comparison to K8s:**
*   **Our Scheduler:** Blind Round-Robin.
*   **K8s Scheduler:** Multi-stage filtering (Predicates) and Scoring (Priorities). Checks CPU/Mem request vs Node Capacity, Taints, Tolerations, Affinity.

### 2.4 The Kubelet Agent: Edge-Triggered Reconciliation

**File:** `my-kube/agent/agent.go`

The Agent is the "Muscle". It wraps the "Brain" (`my-runc`).

**The Sync Loop (`sync()`):**
1.  **Download Desired State:** Fetch list of Pods assigned to me.
2.  **Check Local State:** What am I currently running? (In our simple version, we track this in a `knownPods` map. In reality, we would query the Docker Daemon/CRI).
3.  **Reconcile:**
    *   If in Desired but not Local -> `runtime.RunPod()`.
    *   (Future) If in Local but not Desired -> `runtime.StopPod()`.

This is **Level-Triggered** logic (mostly). We check the *state*, not just events. If the server crashes and comes back, the next poll syncs everything correctly.

---

## Chapter 3: Critical Debugging Case Studies

These are real issues encountered during the development of this project.

### 3.1 The "Invalid Gateway" Subnet Error

**Error:**
```text
Error: Nexthop has invalid gateway.
```

**Scenario:**
We configured the container IP as `10.244.0.100` and the Gateway as `10.244.0.1`.

**The Root Cause:**
When you configure an IP without a subnet mask (CIDR) in Linux `ip` command, it defaults to `/32` (255.255.255.255).
*   **Container View:** "My IP is 10.244.0.100. My network size is 1 address. Everyone else is an alien."
*   **Action:** Container tries to reach Gateway `10.244.0.1`.
*   **Check:** "Is `10.244.0.1` in my network?" -> **NO**.
*   **Result:** "I need a gateway to reach... my gateway." (Recursive failure).

**The Fix:**
We changed the IP assignment to `10.244.0.100/16`.
*   **Container View:** "My IP is 10.244.0.100. My network is `10.244.0.0 - 10.244.255.255`."
*   **Check:** "Is `10.244.0.1` in my network?" -> **YES**.
*   **Result:** ARP request sent. Connection established.

### 3.2 The "Ping 8.8.8.8" Failure (NAT)

**Scenario:**
Container could ping the Bridge (`10.244.0.1`) but not Google (`8.8.8.8`). `tcpdump` showed packets leaving the VM but never coming back.

**The Root Cause:**
We forgot **Network Address Translation (NAT)**.
The external internet does not route Private RFC1918 addresses (`10.x.x.x`). When the packet reached Google, Google's server saw a Source IP of `10.244.0.100`. It attempted to reply, but its local router dropped the packet as "bogon" (invalid internet traffic).

**The Fix:**
We enabled **IP Masquerading** on the Host VM.
This forces the Host VM to replace the packet's source address with its *own* valid LAN address before sending it out. It acts as a proxy for the container.