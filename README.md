# Mini-Kube Project

This project aims to demystify Docker and Kubernetes by building them from scratch. It includes:

1. A Container Runtime (`my-runc`) - A simplified version of runc that implements containerization features
2. An Orchestrator (`my-kube`) - A simplified version of Kubernetes that manages containers

## Project Structure

```
mini-kube/
├── README.md
└── my-runc/
    ├── main.go
    └── go.mod
```

## Implementation Progress

### Tasks Completed:
1. Research and understand Linux kernel concepts: namespaces, cgroups, and pivot_root
   - Successfully understood the fundamental concepts that underpin containerization
   - Namespaces provide process isolation (PID, network, mount namespaces)
   - Cgroups enable resource limiting and accounting (CPU, memory, etc.)
   - Pivot_root allows changing the root filesystem for isolation
2. Setup development environment with Go and necessary tools
   - Installed and configured Go development environment (version 1.24.4 in VM)
   - Installed necessary system packages including build-essential
   - Set up proper Go module structure
3. Install and configure Lima VM
   - Created Ubuntu VM with proper configuration
   - Configured VM with sufficient resources (4 CPU cores, 4GB RAM)
   - Verified that the VM has necessary kernel features for containerization
4. Create basic Go project structure for my-runc
   - Created directory structure for my-runc project
   - Initialized Go module with proper dependencies
   - Created basic main.go file with command-line interface
   - Implemented initial build and test capabilities
5. Implement basic chroot functionality in Go
   - Created a working executable that can parse command-line arguments
   - Built and tested the initial my-runc binary successfully
   - Verified the executable can be run in the VM environment

### Next Steps:
1. Add namespace isolation using syscall.CLONE_NEWPID, CLONE_NEWNS, CLONE_NEWNET
   - Implement process isolation with PID, network, and mount namespaces
   - Test namespace isolation functionality
2. Implement cgroups setup to limit RAM and CPU
   - Create cgroup management functions
   - Implement resource limits for containers
   - Test resource limiting functionality
3. Implement pivot_root functionality
   - Implement root filesystem changes for container isolation
   - Test filesystem isolation
4. Create test cases to verify container isolation
   - Develop automated tests for isolation features
   - Validate that containers are properly isolated
5. Implement basic my-kube orchestrator API server
   - Create API server to accept pod specifications
   - Implement basic pod scheduling
6. Create simple scheduler logic for pod placement
   - Implement basic scheduling algorithm
   - Test with multiple pods
7. Implement kubelet that calls my-runc to start processes
   - Create kubelet that manages container lifecycle
   - Integrate with my-runc for actual container execution
8. Design pod specification format
   - Define structure for pod specifications
   - Create validation for pod configurations
9. Implement pod lifecycle management (create, start, stop)
   - Implement full pod lifecycle management
   - Add monitoring capabilities
10. Test the full container orchestration workflow
    - End-to-end testing of container creation and management
    - Integration testing of all components
11. Add monitoring and logging capabilities
    - Implement logging for container operations
    - Add monitoring features
12. Document the implementation and testing process
    - Create comprehensive documentation
    - Document design decisions and implementation details
13. Optimize and refactor code for performance and clarity
    - Refactor for maintainability
    - Optimize for performance
14. Create comprehensive README with usage instructions
    - Add detailed usage instructions
    - Document all features and capabilities
15. Validate all components work together as a complete system
    - Full system integration testing
    - Performance and correctness validation

## Getting Started

1. Ensure you're running as root in the VM (required for containerization features)
2. Navigate to the my-runc directory
3. Build with: `go build -o my-runc .`

## Current Status

The project is in the early stages of implementation. We have successfully:
- Set up a proper development environment with Lima VM
- Created the basic project structure for my-runc
- Implemented a working command-line interface
- Built and tested the initial my-runc executable

The next phase will involve implementing actual containerization features using Linux kernel capabilities.

## Technical Details

### Linux Kernel Features Used

1. **Namespaces** - Isolate processes and their view of the system:
   - PID namespaces (separate process ID spaces)
   - Network namespaces (separate network interfaces)
   - Mount namespaces (separate filesystem views)

2. **Cgroups** - Control resource usage:
   - CPU limiting and accounting
   - Memory limits and accounting
   - Process group management

3. **Pivot_root** - Change root filesystem:
   - Create isolated filesystem environments
   - Implement container root filesystem isolation

### Design Philosophy

- Follow the Open Container Initiative (OCI) specifications
- Use minimal dependencies to understand core concepts
- Focus on correctness and educational value over production features
- Implement features incrementally with proper testing

## Development Approach

This project follows an incremental approach:
1. Start with basic structure and functionality
2. Add one containerization feature at a time
3. Test each feature thoroughly before moving to the next
4. Document implementation decisions and lessons learned

The implementation is focused on understanding how containerization works under the hood, rather than creating a production-ready system.