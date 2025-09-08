# sharing_renter_system

This document provides instructions on how to set up and manage this project using Docker Compose.

### Git Installation

```bash
sudo apt update
sudo apt install git -y
```

### Project Codebase Cloning

Clone the project into the `/var/www` directory.

```bash
sudo git clone https://github.com/zic9494/SharingRenterSystem.git
```

## Docker Compose Instructions

### Notice

Your WSL version must be greater than version 2, and all command are word at Project root directory

### Start Services

Select the appropriate Docker Compose configuration file to start the services based on your development environment and requirements.

**Docker start command:**
```bash
docker compose -f docker/docker-compose.yml up --build -d
# --build for rebuild docker
# -d for run at background
```

*   **Are you use WSL and have problem at CLI？**
    ```bash
    ls -l /var/run/docker.sock || true
    readlink -f /var/run/docker.sock || true
    # must be /var/run/docker.sock

    # check connection to windows
    docker info | grep -i -E 'Server Version|Docker Desktop|Context' || echo "no server"
    # Operating System: Docker Desktop
    ```
*   **Fix the problem**
    Open docker dosktop → setting → Resources → WSL integration → Enable integration with additional distros → Ubuntu: on

### Shutdown Service

**Docker shutdown command**
```bash
docker compose -f docker/docker-compose.yml down -v
# -v for delete all volumes
```

### Stop Service
**Docker Stop command**
```bash
docker compose -f docker/docker-compose.yml stop
```

### Restart Service
**Docker Restart command**
```bash
docker compose -f docker/docker-compose.yml restart
```
