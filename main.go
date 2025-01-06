package main

import (
    "fmt"
    "log"
    "sync"
    "os"

    // go-plugins-helpers for volume
    "github.com/docker/go-plugins-helpers/volume"
)

// myDriver implements the volume.Driver interface
type myDriver struct {
    // A simple in-memory map: volumeName -> mountPath
    // Use a mutex or sync.Map to protect concurrent map access
    volumes map[string]string
    m       sync.Mutex
}

func newMyDriver() *myDriver {
    return &myDriver{
        volumes: make(map[string]string),
    }
}

func (d *myDriver) Create(req *volume.CreateRequest) error {
    d.m.Lock()
    defer d.m.Unlock()

    if _, exists := d.volumes[req.Name]; exists {
        // If the volume already exists, do nothing or return an error.
        return nil
    }

    mountPath := fmt.Sprintf("/tmp/%s", req.Name)

    // Create the directory on the host so Docker can mount it.
    if err := os.MkdirAll(mountPath, 0755); err != nil {
        return err
    }

    d.volumes[req.Name] = mountPath
    log.Printf("Created volume: %s at path %s\n", req.Name, mountPath)
    return nil
}

// Create is called when Docker wants to create a volume.
// You can store metadata or provision real storage here.
//func (d *myDriver) Create(req *volume.CreateRequest) error {
//    d.m.Lock()
//    defer d.m.Unlock()
//
//    _, exists := d.volumes[req.Name]
//    if exists {
//        // Already exists, do nothing or return an error
//        return nil
//    }

    // For demonstration, just store a path. In real usage, you might create
    // a directory on the host or initiate a connection to network storage.
//    mountPath := fmt.Sprintf("/tmp/%s", req.Name)
///    d.volumes[req.Name] = mountPath

//    log.Printf("Created volume: %s at path %s\n", req.Name, mountPath)
//    return nil
//}

// Remove is called when Docker wants to remove a volume.
func (d *myDriver) Remove(req *volume.RemoveRequest) error {
    d.m.Lock()
    defer d.m.Unlock()

    delete(d.volumes, req.Name)
    log.Printf("Removed volume: %s\n", req.Name)
    return nil
}

// Mount is called when a container starts and the volume is requested.
// Docker wants you to return a path on the host filesystem that it can bind-mount.
func (d *myDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
    d.m.Lock()
    defer d.m.Unlock()

    mountPath, exists := d.volumes[req.Name]
    if !exists {
        return nil, fmt.Errorf("volume %s not found", req.Name)
    }

    // In a real plugin, you'd do actual mounting logic here (e.g. `mount -t nfs ...`).
    // For demonstration, we just rely on the path we stored.

    log.Printf("Mounting volume: %s at path %s (ID: %s)\n", req.Name, mountPath, req.ID)
    return &volume.MountResponse{Mountpoint: mountPath}, nil
}

// Unmount is called when a container using the volume stops or no longer needs the volume.
func (d *myDriver) Unmount(req *volume.UnmountRequest) error {
    d.m.Lock()
    defer d.m.Unlock()

    mountPath, exists := d.volumes[req.Name]
    if !exists {
        return fmt.Errorf("volume %s not found", req.Name)
    }

    log.Printf("Unmounting volume: %s from path %s (ID: %s)\n", req.Name, mountPath, req.ID)
    // In a real plugin, you might do `umount(mountPath)`.
    return nil
}

// List returns all volumes that this driver knows about.
func (d *myDriver) List() (*volume.ListResponse, error) {
    d.m.Lock()
    defer d.m.Unlock()

    var vols []*volume.Volume
    for name, path := range d.volumes {
        vols = append(vols, &volume.Volume{
            Name:       name,
            Mountpoint: path,
        })
    }
    return &volume.ListResponse{Volumes: vols}, nil
}

// Get returns the volume info requested.
func (d *myDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
    d.m.Lock()
    defer d.m.Unlock()

    path, exists := d.volumes[req.Name]
    if !exists {
        return nil, fmt.Errorf("volume %s not found", req.Name)
    }
    vol := &volume.Volume{
        Name:       req.Name,
        Mountpoint: path,
    }
    return &volume.GetResponse{Volume: vol}, nil
}

func (d *myDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
    d.m.Lock()
    defer d.m.Unlock()

    path, ok := d.volumes[req.Name]
    if !ok {
        return nil, fmt.Errorf("volume %s not found", req.Name)
    }

    return &volume.PathResponse{Mountpoint: path}, nil
}

// Capabilities tells Docker which advanced features this driver supports.
// For example, you could say Scope = "global" if the volumes are available
// across multiple hosts in a cluster.
func (d *myDriver) Capabilities() *volume.CapabilitiesResponse {
    return &volume.CapabilitiesResponse{
        Capabilities: volume.Capability{
            Scope: "local",
        },
    }
}

func main() {
    driver := newMyDriver()

    // The go-plugins-helpers library offers a convenience method
    // to create a Unix socket server for your driver.
    h := volume.NewHandler(driver)

    // The first parameter is the "plugin name" (used to create the .sock file),
    // the second is the group. 0 means 'root' by default.
    log.Println("Starting my-volume-plugin on /run/docker/plugins/my-volume-plugin.sock ...")
    err := h.ServeUnix("my-volume-plugin", 0)
    if err != nil {
        log.Fatalf("Error serving volume plugin: %v", err)
    }
}
