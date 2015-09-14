package nvim

import "net/rpc"

// Vim object itself is not registered on the nvim api as a custom type. so define here
// it's called Vim and not NVim for now. as we just generate the struct names based on the rpc method names
// eg: vim_get_current_buffer
type Vim struct {
	// Id     Identifier
	client *rpc.Client
}
