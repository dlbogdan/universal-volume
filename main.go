package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	// go-plugins-helpers for volume
	"github.com/docker/go-plugins-helpers/volume"
)

func isMountpoint(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("Failed to stat path: %w", err)
	}

	parentPath := filepath.Join(path, "..")
	parentStat, err := os.Stat(parentPath)
	if err != nil {
		return false, fmt.Errorf("Failed to stat parent path: %w", err)
	}

	// Get device numbers for path and its parent
	statSys := stat.Sys().(*syscall.Stat_t)
	parentStatSys := parentStat.Sys().(*syscall.Stat_t)

	// Compare device numbers and inode numbers
	isMount := statSys.Dev != parentStatSys.Dev || statSys.Ino == parentStatSys.Ino
	return isMount, nil
}

// myDriver implements the volume.Driver interface
type myDriver struct {
	rootPath string
}

func newMyDriver() *myDriver {
	envRootPath := os.Getenv("ROOT_PATH")
	envScope := os.Getenv("SCOPE") // global or local

	if envScope == "" {

		// Default to global scope if SCOPE is not set
		log.Println("SCOPE not set, using global")
		envScope = "global"
	}

	if envRootPath == "" {
		// Default to /var/lib/myvolplugin if ROOT_PATH is not set
		log.Println("ROOT_PATH not set, using /var/lib/myvolplugin")
		envRootPath = "/var/lib/myvolplugin"
	}
	if envScope == "global" {
		var isMount bool
		var err error
		log.Printf("Scope is global. Assuming the filesystem is distributed. Checking if %s is a mountpoint\n", envRootPath)
		for i := 0; i < 10; i++ {
			isMount, err = isMountpoint(envRootPath)
			if err == nil && isMount {
				break
			}
			log.Printf("Error checking mountpoint (attempt %d): %v, isMount: %v\n", i+1, err, isMount)
			time.Sleep(1 * time.Second)
		}

		if err != nil || !isMount {
			log.Fatalf("Failed to verify mountpoint after 10 attempts: %v, isMount: %v\n", err, isMount)
		}
	}
	envRootPath = filepath.Join(envRootPath, "volumes")
	return &myDriver{rootPath: envRootPath}
}

func (d *myDriver) Create(req *volume.CreateRequest) error {
	fullPath := filepath.Join(d.rootPath, req.Name)

	// If the folder exists, do nothing.
	// (Alternatively, return an error if you want to forbid overwriting.)
	if _, err := os.Stat(fullPath); err == nil {
		return nil
	}

	// Create the directory
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory for volume %s: %v", req.Name, err)
	}

	log.Printf("Created volume: %s\n", req.Name)
	return nil
}

func (d *myDriver) Remove(req *volume.RemoveRequest) error {
	fullPath := filepath.Join(d.rootPath, req.Name)

	// If it doesn't exist, do nothing
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil
	}

	// In a real plugin, you might check if it's still in use before removing.
	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to remove volume folder: %v", err)
	}
	log.Printf("Removed volume: %s\n", req.Name)

	return nil
}

func (d *myDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	fullPath := filepath.Join(d.rootPath, req.Name)
	log.Printf("Mounting volume: %s, id: %s\n", req.Name, req.ID)
	// Just verify it exists

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("volume %s not found", req.Name)
	}
	// Return the path so Docker can do a bind mount
	return &volume.MountResponse{Mountpoint: fullPath}, nil
}

func (d *myDriver) Unmount(req *volume.UnmountRequest) error {
	// For local directories, there's nothing to unmount, but let's check existence
	fullPath := filepath.Join(d.rootPath, req.Name)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil
	}
	log.Printf("Unmounting volume: %s (ID: %s)\n", req.Name, req.ID)
	// If you have something more advanced, handle it here
	return nil
}

// List finds all directories in d.rootPath and reports them as volumes.
func (d *myDriver) List() (*volume.ListResponse, error) {
	// We'll store the discovered volumes in 'vols'
	var vols []*volume.Volume

	entries, err := os.ReadDir(d.rootPath)
	if err != nil {
		// If the rootPath does not exist or can't be read, log it and return no volumes
		// or return an error. Your choice depends on your plugin’s design.
		return &volume.ListResponse{Volumes: vols}, nil
	}

	for _, entry := range entries {
		// Only treat subdirectories as volumes
		if entry.IsDir() {
			volName := entry.Name()
			mountPath := filepath.Join(d.rootPath, volName)
			vols = append(vols, &volume.Volume{
				Name:       volName,
				Mountpoint: mountPath,
			})
		}
	}

	return &volume.ListResponse{Volumes: vols}, nil
}

func (d *myDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	fullPath := filepath.Join(d.rootPath, req.Name)

	// Check if the directory (i.e., volume) exists
	fi, err := os.Stat(fullPath)
	if os.IsNotExist(err) || (err == nil && !fi.IsDir()) {
		return nil, fmt.Errorf("volume %s not found", req.Name)
	} else if err != nil {
		return nil, err
	}

	// Directory exists, so return the volume info
	vol := &volume.Volume{
		Name:       req.Name,
		Mountpoint: fullPath,
	}
	return &volume.GetResponse{Volume: vol}, nil
}

func (d *myDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	// Build the full path by joining ROOT_PATH and the volume name
	fullPath := filepath.Join(d.rootPath, req.Name)

	// Check if the directory exists on disk
	fi, err := os.Stat(fullPath)
	if os.IsNotExist(err) || (err == nil && !fi.IsDir()) {
		return nil, fmt.Errorf("volume %s not found", req.Name)
	} else if err != nil {
		return nil, err
	}

	// Directory exists, so return its path
	return &volume.PathResponse{Mountpoint: fullPath}, nil
}

// Capabilities tells Docker which advanced features this driver supports.
// For example, you could say Scope = "global" if the volumes are available
// across multiple hosts in a cluster.
func (d *myDriver) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{
			Scope: "global",
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
	//ceph-fuse --client_fs swarm_cephfs /mnt/swarm_cephfs/

	// if _, err := os.Stat("/mnt/swarm_cephfs/volumes"); os.IsNotExist(err) {
	// 	log.Println("Mounting cephfs")
	// 	exec.Command("ceph-fuse", "--client_fs", "swarm_cephfs", "/mnt/swarm_cephfs").Run()
	// }

	err := h.ServeUnix("my-volume-plugin", 0)
	if err != nil {
		log.Fatalf("Error serving volume plugin: %v", err)
	}
}
