WIP 
# universal-volume

A **Docker volume plugin** that automatically enumerates and manages directories under a user-defined **root path** on the host. Designed to be universally simple—no databases, no complicated setup—just point it to a root directory, and any subdirectory becomes a usable Docker volume.


## Table of Contents

1. [Features](#features)  
2. [How It Works](#how-it-works)  
3. [Installation & Setup](#installation--setup)  
4. [Configuration](#configuration)  
5. [Usage](#usage)  
6. [Development](#development)  
7. [License](#license)

---

## Features

- **Configurable Root Path**: Define `ROOT_PATH` (e.g., `/mnt/univol`) where all volumes reside.  
- **Automatic Enumeration**: On `docker volume ls` or plugin queries, the plugin scans the root path for subdirectories and treats each as a volume.  
- **Simple Local/mounted distributed Storage**: No external DB or advanced network mount logic—just host folders.  
- **Persistent**: Volumes persist across Docker or plugin restarts (as long as folders remain in `ROOT_PATH`).  
- **Easy Installation**: Distributed as a Docker-managed plugin.  

---

## How It Works

1. **Root Directory**: You specify a `ROOT_PATH` (e.g., `/mnt/univol`).  
2. **Create**: When `docker volume create -d universal-volume --name myvol` is called, the plugin creates a folder under the specified root path (e.g., `/var/lib/universal-volume/myvol`).  
3. **Mount**: The plugin returns that path as the `Mountpoint`, which Docker bind-mounts into your container.  
4. **Enumerate**: When Docker asks for a volume **list**, the plugin reads subfolders of the root path and reports each one as a volume.  
5. **Remove**: A `docker volume rm myvol` call removes the corresponding folder on disk.

---

## Installation & Setup

1. **Build the Plugin Locally** (optional, if you want to tweak the source):
   ```bash
   git clone https://github.com/dlbogdan/universal-volume.git
   cd universal-volume
   docker build -t universal-volume-build .
   ```

2. **Create the Managed Plugin**:
   - Export the container filesystem to `rootfs/`:
     ```bash
     CONTAINER_ID=$(docker create universal-volume-build)
     mkdir -p rootfs
     docker export "$CONTAINER_ID" | tar -x -C rootfs
     docker rm -v "$CONTAINER_ID"
     ```
   - Ensure you have a **`config.json`** at the top-level (beside `rootfs/`), defining entrypoint/capabilities/etc.
   - Then create the plugin:
     ```bash
     docker plugin create dlbogdan/universal-volume:latest .
     ```

3. **Push the Plugin** (optional, to share it publicly):
   ```bash
   docker plugin push dlbogdan/universal-volume:latest
   ```

4. **Install on Another Host**:
   ```bash
   docker plugin install --grant-all-permissions dlbogdan/universal-volume:latest
   ```
   (You can now configure or enable the plugin in the next step.)

---

## Configuration

You can configure the plugin by setting environment variables via **`docker plugin set`**. Two typical variables are:

- **`ROOT_PATH`**: Path on the host where volumes will be created (default might be `/var/lib/universal-volume`).  
- **`SCOPE`**: Plugin scope (`local` or `global`).

For example:

```bash
docker plugin set <your-dockerhub-username>/universal-volume:latest \
  ROOT_PATH=/mnt/univol \
  SCOPE=global
```

Then enable the plugin:

```bash
docker plugin enable dlbogdan/universal-volume:latest
```

---

## Usage

1. **Create a Volume**:
   ```bash
   docker volume create -d universal-volume --name mytest
   ```
   - Creates a directory under your configured `ROOT_PATH`, e.g. `/mnt/univol`.

2. **Run a Container**:
   ```bash
   docker run --rm -it -v mytest:/data busybox sh
   ```
   - The plugin returns the local directory as the mount path, and Docker bind-mounts it to `/data`.

3. **List Volumes**:
   ```bash
   docker volume ls
   ```
   - The plugin enumerates subdirectories in `ROOT_PATH`, reporting each as a volume.

4. **Remove a Volume**:
   ```bash
   docker volume rm mytest
   ```
   - Deletes the corresponding directory from disk.

---

## Development

- **Languages**: Primarily Go, using the [go-plugins-helpers](https://github.com/docker/go-plugins-helpers) library.
- **Local Testing**:
  1. `go build -o universal-volume .` to build the binary.  
  2. Run it manually with `sudo ./universal-volume` for direct testing without a managed plugin.  
  3. If Docker doesn’t automatically recognize the socket, place a JSON file like `/etc/docker/plugins/universal-volume.json` with:
     ```json
     {
       "Name": "universal-volume",
       "Addr": "unix:///run/docker/plugins/universal-volume.sock"
     }
     ```
  4. Use `docker volume create -d universal-volume ...` to test locally.
- **Contributions**:  
  1. Fork the repo  
  2. Open a Pull Request  
  3. We’ll review & merge changes

---

## License

This project is licensed under the [MIT License](./LICENSE) (or whichever license you choose). See the [LICENSE](./LICENSE) file for details.

---

**Enjoy universal-volume!** If you have questions or encounter issues, open an [issue on GitHub](https://github.com/<your-username>/universal-volume/issues). Contributions and feedback are always welcome.
