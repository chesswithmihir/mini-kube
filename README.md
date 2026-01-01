# Mini-Kube: The Definitive Systems Textbook Project

This project aims to **demystify the "magic" of Docker and Kubernetes** by building foundational components from scratch. It provides a hands-on approach to understanding the fundamental Linux kernel concepts that power modern containerization and orchestration. By building simplified versions of `runc` (container runtime) and Kubernetes (orchestrator), we explore the low-level mechanics that often remain hidden behind high-level tools.

## Project Vision & Goal

**Vision:** To prove that Kubernetes is simply a **Distributed System built on top of File Operations**. By representing Namespaces, Cgroups, and Pod States as files and HTTP requests, we make the complex simple.

**Goal:** To build a **production-principled** container orchestrator. We strip away the "magic" of `docker run` and `kubectl apply` to reveal the raw Linux primitives: `clone`, `pivot_root`, `veth` pairs, and `iptables`. By the end, you will not just *use* Kubernetes; you will understand the syscalls that make it possible.

---

## Part 1: `my-runc` - The Container Runtime

`my-runc` is our simplified implementation of an OCI (Open Container Initiative) compliant runtime, analogous to `runc`. It directly interacts with the Linux kernel to create and manage isolated container environments.

### Core Capabilities: Linux Kernel Primitives in Action

`my-runc` leverages several key Linux kernel features to provide container isolation and resource management:

*   **Linux Namespaces (Isolation):** Provides each container with its own isolated view of system resources. `my-runc` implements:
    *   **PID Namespace:** Independent Process ID space.
    *   **Mount Namespace:** Isolated filesystem hierarchy.
    *   **UTS Namespace:** Independent hostname.
    *   **IPC Namespace:** Isolated Inter-Process Communication.
    *   **Network Namespace:** Isolated network stack (devices, IPs, routing).
    *   **User Namespace:** Isolated User/Group IDs, allowing container `root` to map to an unprivileged host user for enhanced security.
*   **Control Groups (Cgroups) (Resource Limits):** Manages and limits system resources (like CPU, memory, I/O) allocated to containers. `my-runc` supports both Cgroups v1 and v2, primarily demonstrating memory limiting.
*   **`pivot_root` (Filesystem Isolation):** Securely changes the root filesystem of the container, completely detaching it from the host's filesystem.

### Advanced Implementation Details

*   **The "Re-Execution" Pattern:** `my-runc` employs a sophisticated parent-child re-execution model. The main `my-runc` process launches itself again in a "child" mode within newly created namespaces. This child then performs internal setup (Cgroups, `pivot_root`) before executing the user's command.
*   **Pipe Synchronization (Parent-Child IPC):** A critical aspect of the re-execution pattern is the use of a synchronization pipe. The parent process creates a pipe and passes its read-end to the child. The child performs internal setup, then blocks on this pipe. The parent, after launching the child, performs external setup (like network configuration) on the child's new namespaces, then signals the child via the pipe. This ensures precise ordering and prevents race conditions in the container's environment setup.
*   **Functional Networking:** Unlike a "roadmap" item, `my-runc` includes a functional single-host bridge networking implementation. It uses `veth` pairs, bridge devices (`my-bridge0`), and `iptables` for NAT/masquerading, allowing containers to have their own IP addresses and reach the internet.

### Usage

**Building the `my-runc` binary:**

Navigate to the root directory and run:
```bash
make my-runc
```

**Executing an isolated process with Networking:**

This command demonstrates `my-runc`'s core capabilities:
```bash
# This command:
# 1. Spawns a new process in a private PID/Net/Mount namespace.
# 2. Automatically creates 'my-bridge0' on your host.
# 3. Injects a virtual ethernet cable into the container.
# 4. Sets up NAT so the container can reach the internet.
sudo ./bin/my-runc run --ip 10.244.0.100 sh -c "ip addr && ping -c 1 8.8.8.8"
```

**Other `my-runc` commands:**

*   **`./bin/my-runc run <command>`:** Runs a command in an isolated environment (without networking by default).
*   **`./bin/my-runc spec`:** (Placeholder) Generates a container specification.
*   **`./bin/my-runc version`:** Displays `my-runc`'s version.

### Testing

Comprehensive tests validate `my-runc`'s Linux-specific features, including hostname, PID, and filesystem isolation, as well as cgroup memory limits.

To run unit tests:
```bash
make test
```

To run the full system End-to-End suite (which mocks the runtime):
```bash
make test-e2e
```

### Running on macOS (via Lima)

Since macOS (Darwin kernel) does not support Linux namespaces or Cgroups, `my-runc` can be developed and run within a lightweight Linux VM using `limactl` (Lima).

1.  **Start a Linux VM:** `limactl start ubuntu`
2.  **Enter the VM:** `limactl shell ubuntu`
3.  **Copy Code:** Copy the project to a local directory in the VM (shared folders can have limitations):
    ```bash
    cp -r /path/to/mini-kube/my-runc ~/my-runc-playground
    cd ~/my-runc-playground
    ```
4.  **Build and Run:**
    ```bash
    go build -o my-runc .
    sudo ./my-runc run bash
    ```

---

## Part 2: `my-kube` - The Orchestrator

`my-kube` is a **functional, albeit simplified, implementation of a Kubernetes-like orchestrator**. It manages `my-runc` containers across a simulated cluster, demonstrating the core principles of distributed systems and desired state reconciliation.

### Architecture: The Mini-Cloud ("Manager" vs. "Worker")

The `my-kube` architecture simulates a cluster using a "Manager" (Control Plane) and "Worker" nodes.

*   **The Control Plane (`my-kube-server`) - The "Manager":**
    *   **Role:** The brain of the cluster. It defines and maintains the "Desired State" of applications.
    *   **Components:**
        *   **API Server:** A Go HTTP server that stores the cluster's state (e.g., information about Pods, Nodes). It exposes endpoints for creating and querying resources.
        *   **Scheduler:** A component that continuously watches for "Pending Pods" (desired workloads without assigned execution locations) and assigns them to available "Worker" nodes based on defined policies (e.g., resource availability).
*   **The Worker Node (`my-kubelet`) - The "Kitchen":**
    *   **Role:** The muscle of the cluster. It executes the workloads assigned by the Control Plane.
    *   **Components:**
        *   **Kubelet Agent:** An agent that runs on each worker node. It constantly polls the API Server to discover what Pods it should be running.
        *   **`my-runc` Integration:** When assigned a Pod, the Kubelet calls `my-runc` to spin up the container locally. It then monitors the container's health and reports its status back to the API Server.
*   **The Workload (The "App"):**
    *   These are the actual user applications running inside `my-runc` containers on the worker nodes. `my-kube` is designed to orchestrate long-running services, such as simple HTTP servers, to demonstrate cluster networking.

### Core Orchestration Principles

`my-kube` embodies the core Kubernetes principle of **Desired State vs. Actual State** reconciliation:

*   **Desired State:** What the user *wants* the cluster to look like (e.g., "I want 3 copies of Nginx running"). This state is stored in the API Server.
*   **Actual State:** The current reality of the cluster (e.g., "Only 2 copies are running because one crashed").
*   **Reconciliation Loop:** The Scheduler and Kubelet agents constantly observe both states. If they don't match, they take action to bring the Actual State in line with the Desired State (e.g., starting another Nginx instance). This makes the system **self-healing**.

### Cluster Networking

`my-kube` builds upon `my-runc`'s networking capabilities to enable communication between containers and nodes. This allows for the deployment of networked applications (like web servers) where components can reach each other across the cluster, a fundamental requirement for distributed applications.

### Usage

`my-kube` is designed to run across multiple Linux VMs (e.g., using Lima).

*   **Start `my-kube-server`:**
    ```bash
    cd my-kube/server
    go run main.go
    ```
*   **Start `my-kubelet` agent on worker nodes:**
    ```bash
    cd my-kube/agent
    go run main.go --api-server-ip <SERVER_IP>
    ```
    (Note: Specific commands for deploying Pods via the API are not yet documented here but are handled by the server's API endpoints.)

---

## Multi-VM Cluster Setup: Running `my-kube` with `limactl`

`my-kube` is designed to run in a distributed fashion across multiple nodes. To achieve this on macOS, we use **Lima** with a shared network bridge (`socket_vmnet`) to allow VMs to communicate.

---

**Step 0: Prepare Your Host Machine (One-time setup)**

1.  **Install Prerequisites:**
    ```bash
    brew install lima socket_vmnet
    ```

2.  **Configure `socket_vmnet` (Requires sudo):**
    ```bash
    sudo cp "$(brew --prefix socket_vmnet)/bin/socket_vmnet" /opt/socket_vmnet/bin/socket_vmnet
    limactl sudoers | sudo tee /etc/sudoers.d/lima
    ```

3.  **Clean up stale sockets (if any):**
    ```bash
    sudo rm -f /private/var/run/lima/socket_vmnet.shared
    ```

---

**Step 1: Create and Start Lima VMs**

We'll use a specific configuration (`cluster-vm.yaml`) to enable the shared network.

1.  **Create `cluster-vm.yaml`:**
    ```yaml
    vmType: "vz"
    cpus: 1
    memory: "1GiB"
    disk: "10GiB"
    mounts:
      - location: "~"
        writable: true
    networks:
      - lima: shared
    images:
    - location: "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img"
      arch: "x86_64"
    - location: "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-arm64.img"
      arch: "aarch64"
    ```

2.  **Create and Start VMs:**
    ```bash
    limactl create --name master-node cluster-vm.yaml
    limactl create --name worker-node-1 cluster-vm.yaml
    limactl create --name worker-node-2 cluster-vm.yaml
    
    limactl start master-node worker-node-1 worker-node-2
    ```

---

**Step 2: Build and Distribute Binaries**

Since we are on macOS but targeting Linux VMs, we must **cross-compile**. The binaries will be available to the VMs via the shared mount.

1.  **Build for Linux:**
    ```bash
    GOOS=linux GOARCH=arm64 make all
    ```

---

**Step 3: Get the Cluster IPs**

Verify that all nodes are on the `192.168.105.x` network.

1.  **Check IP on Master:**
    ```bash
    limactl shell master-node ip a show lima0
    ```
    *(Note this IP, e.g., `192.168.105.2`).*

---

**Step 4: Start the Cluster**

Open 3 terminal tabs.

1.  **Terminal 1 (Master Node):**
    ```bash
    limactl shell master-node
    cd /path/to/mini-kube
    ./bin/my-kube-server --port 8080
    ```

2.  **Terminal 2 (Worker 1):**
    ```bash
    limactl shell worker-node-1
    cd /path/to/mini-kube
    sudo ./bin/my-kubelet --node worker-node-1 --server http://192.168.105.2:8080 --runc ./bin/my-runc
    ```

3.  **Terminal 3 (Worker 2):**
    ```bash
    limactl shell worker-node-2
    cd /path/to/mini-kube
    sudo ./bin/my-kubelet --node worker-node-2 --server http://192.168.105.2:8080 --runc ./bin/my-runc
    ```

---

**Step 5: Deploy and Verify a Workload**

1.  **Prepare `pod.json`:**
    ```json
    {
      "id": "my-web-server",
      "name": "my-web-server-pod",
      "command": ["python3", "-m", "http.server", "8000"],
      "pod_ip": "10.244.0.100",
      "status": "Pending"
    }
    ```

2.  **Submit Pod:**
    ```bash
    limactl shell master-node curl -X POST -H "Content-Type: application/json" -d @pod.json http://localhost:8080/pods
    ```

3.  **Verify Reachability:**
    From the worker node, curl the container IP:
    ```bash
    limactl shell worker-node-1 curl -I http://10.244.0.100:8000
    ```

---

**Step 6: Clean Up**

```bash
limactl stop master-node worker-node-1 worker-node-2
limactl delete master-node worker-node-1 worker-node-2
```

---

## Development Notes

This project is developed primarily for educational purposes. Its goal is to provide a deep understanding of containerization and orchestration concepts by building simplified versions of widely used tools. It is not intended for production use, as it lacks robustness, security hardening, and full feature sets found in production-grade runtimes and orchestrators.
