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

Navigate to the `my-runc` directory and run:
```bash
cd my-runc && go build -o my-runc .
```

**Executing an isolated process with Networking:**

This command demonstrates `my-runc`'s core capabilities:
```bash
# This command:
# 1. Spawns a new process in a private PID/Net/Mount namespace.
# 2. Automatically creates 'my-bridge0' on your host.
# 3. Injects a virtual ethernet cable into the container.
# 4. Sets up NAT so the container can reach the internet.
sudo ./my-runc run --ip 10.244.0.100 sh -c "ip addr && ping -c 1 8.8.8.8"
```

**Other `my-runc` commands:**

*   **`my-runc run <command>`:** Runs a command in an isolated environment (without networking by default).
*   **`my-runc spec`:** (Placeholder) Generates a container specification.
*   **`my-runc version`:** Displays `my-runc`'s version.

### Testing

Comprehensive tests validate `my-runc`'s Linux-specific features, including hostname, PID, and filesystem isolation, as well as cgroup memory limits.

To run tests (on a Linux system):
```bash
cd my-runc
sudo go test -v ./...
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

`my-kube` is designed to run in a distributed fashion across multiple nodes. We'll set up **3 Lima VMs**:
*   **1 Control Plane Node (master-node):** Will run `my-kube-server`.
*   **2 Worker Nodes (worker-node-1, worker-node-2):** Will run `my-kubelet` agents.

This setup simulates a small Kubernetes cluster.

---

**Step 0: Prepare Your Host Machine (One-time setup)**

Make sure `limactl` is installed and working. You should also ensure you have a `mini-kube` directory on your host that contains all the `my-runc` and `my-kube` code.

---

**Step 1: Create and Start Lima VMs**

We'll create three Ubuntu VMs. For `my-kube` networking, it's essential that these VMs can communicate. Lima's default networking often suffices for VMs on the same host.

1.  **Create `master-node` (1 CPU, 1GiB RAM, 10GiB Disk):**
    ```bash
    limactl create --name master-node --memory=1 --cpus=1 --disk=10
    ```
2.  **Create `worker-node-1` (1 CPU, 1GiB RAM, 10GiB Disk):**
    ```bash
    limactl create --name worker-node-1 --memory=1 --cpus=1 --disk=10
    ```
3.  **Create `worker-node-2` (1 CPU, 1GiB RAM, 10GiB Disk):**
    ```bash
    limactl create --name worker-node-2 --memory=1 --cpus=1 --disk=10
    ```
    *Reminder:* After each `limactl create` command, Lima might open a text editor with the configuration YAML. Just save and close the editor (e.g., `esc`, then `:wq`, then `Enter` for `vi`; `Ctrl+X`, then `Y` to save, then `Enter` for `nano`) to apply the settings.

4.  **Start all VMs:**
    ```bash
    limactl start master-node
    limactl start worker-node-1
    limactl start worker-node-2
    ```
    You can check their status with `limactl list`.

---

**Step 2: Build and Distribute `my-runc` and `my-kube` Binaries**

We need `my-runc` on worker nodes (because `my-kubelet` calls it) and `my-kube-server` on the master, and `my-kubelet` on workers. It's easiest to build all binaries once on your host and then copy them.

1.  **Build all binaries on your host:**
    ```bash
    # From your host's /Users/mihir/git/mini-kube directory
    cd my-runc && go build -o my-runc . && cd ../..
    cd my-kube/server && go build -o my-kube-server . && cd ../..
    cd my-kube/agent && go build -o my-kubelet . && cd ../..
    ```
    You should now have `my-runc`, `my-kube-server`, and `my-kubelet` executables in their respective directories on your host.

2.  **Copy `my-kube-server` to `master-node`:**
    ```bash
    limactl cp my-kube/server/my-kube-server master-node:~/my-kube-server
    ```
    *(You don't strictly need `my-runc` or `my-kubelet` on the master, but no harm in copying if you want a complete set).*

3.  **Copy binaries to `worker-node-1`:**
    ```bash
    limactl cp my-runc/my-runc worker-node-1:~/my-runc
    limactl cp my-kube/agent/my-kubelet worker-node-1:~/my-kubelet
    ```

4.  **Copy binaries to `worker-node-2`:**
    ```bash
    limactl cp my-runc/my-runc worker-node-2:~/my-runc
    limactl cp my-kube/agent/my-kubelet worker-node-2:~/my-kubelet
    ```

---

**Step 3: Get the `master-node` IP Address**

You'll need the IP address of your `master-node` for the worker agents to connect to.

1.  **Shell into `master-node`:**
    ```bash
    limactl shell master-node
    ```
2.  **Get IP address:**
    ```bash
    ip a | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | cut -d/ -f1
    ```
    *Note down this IP address (e.g., `192.168.5.15`). You will use it as `<MASTER_IP>`.*
3.  **Exit `master-node` shell:**
    ```bash
    exit
    ```

---

**Step 4: Start the Control Plane (`my-kube-server`) on `master-node`**

1.  **Shell into `master-node`:**
    ```bash
    limactl shell master-node
    ```
2.  **Run `my-kube-server`:**
    ```bash
    ~/my-kube-server &
    ```
    *This will start the API server and scheduler. You should see output like "Starting my-kube-server (Control Plane)..." and "Listening on :8080...". The `&` puts it in the background so you can still use the terminal. You can safely `exit` this shell after this, or keep it open to see logs.*

---

**Step 5: Start Worker Agents (`my-kubelet`) on `worker-node-1` and `worker-node-2`**

You need to tell each worker where to find the `my-kube-server`.

1.  **Shell into `worker-node-1`:**
    ```bash
    limactl shell worker-node-1
    ```
2.  **Run `my-kubelet`:** (Replace `<MASTER_IP>` with the IP you got in Step 3).
    ```bash
    ~/my-kubelet --api-server-ip http://<MASTER_IP>:8080 --node-id worker-node-1 &
    ```
    *You should see output indicating registration and sync loops. The `--node-id` should match the VM name.*

3.  **Exit `worker-node-1` shell:**
    ```bash
    exit
    ```

4.  **Shell into `worker-node-2`:**
    ```bash
    limactl shell worker-node-2
    ```
5.  **Run `my-kubelet`:** (Replace `<MASTER_IP>` again).
    ```bash
    ~/my-kubelet --api-server-ip http://<MASTER_IP>:8080 --node-id worker-node-2 &
    ```
6.  **Exit `worker-node-2` shell:**
    ```bash
    exit
    ```

*At this point, your `my-kube-server` on `master-node` should be receiving registration requests from `worker-node-1` and `worker-node-2` and they should be visible in its logs. You can re-shell into `master-node` to check `~/my-kube-server` logs.*

---

**Step 6: Deploy a Sample Workload (Python HTTP Server) via the API**

`my-kube` uses its API server to receive workload requests. You'll make a `curl` request to the master-node to create a Pod. This pod will run a simple Python HTTP server within a `my-runc` container.

1.  **From your host machine's terminal**, prepare a JSON file for your pod (e.g., `pod.json`):
    ```json
    {
      "id": "my-web-server",
      "name": "my-web-server-pod",
      "command": ["python3", "-m", "http.server", "8000"],
      "status": "Pending"
    }
    ```
    *Save this as `pod.json` in your host's `mini-kube` directory.*

2.  **Deploy the pod:** (Replace `<MASTER_IP>` with the IP you got in Step 3).
    ```bash
    curl -X POST -H "Content-Type: application/json" -d @pod.json http://<MASTER_IP>:8080/pods
    ```
    *You should receive a JSON response confirming the pod creation. The `my-kube-server` logs on `master-node` should show the pod being created and then assigned to a worker by the scheduler.*
    *The worker node logs (e.g., on `worker-node-1`) should show `my-kubelet` detecting the new pod and calling `my-runc` to start it.*
    *Crucially, `python3` needs to be available in the container's root filesystem (i.e., the default rootfs that `my-runc` is using) for this command to work inside the container.*

---

**Step 7: Verify the Workload**

Once the pod is running, you can try to access the Python HTTP server.

1.  **Find the `PodIP` and assigned `NodeID`:**
    You can `curl` the master's API server again to get the updated status of your pod.
    ```bash
    curl http://<MASTER_IP>:8080/pods
    ```
    *Look for your `my-web-server` pod. It should now have a `NodeID` (e.g., `worker-node-1`) and a `PodIP` (e.g., `10.244.0.100`).*

2.  **Access the web server:**
    *   If the pod was assigned to `worker-node-1`, you'll need the IP address of `worker-node-1`. You can get this like in Step 3, by running `limactl shell worker-node-1` then `ip a`.
    *   From your **host machine**, try to `curl` the worker node's IP on port 8000 (assuming port forwarding is correctly set up by your Lima VM networking, or if `my-runc` configured `iptables` for external access, which it currently does not).

    *Simpler verification*: Log into the worker node that runs the pod, and check if the python server process is running. Also, try to `curl` the `PodIP` from *within* the worker node itself.

    ```bash
    # On your host
    limactl shell <NODE_ID_WHERE_POD_IS_RUNNING>
    # Inside the worker VM
    curl http://<POD_IP>:8000
    # You should see HTML output from the Python server.
    ```

---

**Step 8: Clean Up**

When you're done, clean up your Lima VMs.

1.  **Stop all VMs:**
    ```bash
    limactl stop master-node worker-node-1 worker-node-2
    ```
2.  **Delete all VMs:**
    ```bash
    limactl delete master-node worker-node-1 worker-node-2
    ```

---

## Development Notes

This project is developed primarily for educational purposes. Its goal is to provide a deep understanding of containerization and orchestration concepts by building simplified versions of widely used tools. It is not intended for production use, as it lacks robustness, security hardening, and full feature sets found in production-grade runtimes and orchestrators.
