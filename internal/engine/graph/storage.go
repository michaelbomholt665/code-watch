// # internal/engine/graph/storage.go
package graph

// NodeStorage is the port (interface) for disk-backed graph node persistence.
//
// It decouples the in-memory Graph from any concrete storage layer, allowing
// adapters (e.g. SQLite, in-memory noop) to be swapped without modifying the
// graph package.
//
// A concrete SQLite adapter will live in internal/data/history/graph_storage.go
// and implement this interface. The Graph package only depends on this interface.
type NodeStorage interface {
	// SaveNode persists the module node for the given project and module name.
	// If a node already exists it is overwritten (upsert semantics).
	SaveNode(projectKey, moduleName string, mod *Module) error

	// LoadNode retrieves the module node for the given project and module name.
	// Returns (nil, nil) if the node does not exist.
	LoadNode(projectKey, moduleName string) (*Module, error)

	// QueryEdges returns the list of module names that the given module imports.
	// Returns an empty slice if no edges exist.
	QueryEdges(projectKey, from string) ([]string, error)

	// Close releases any underlying resources (e.g. database connections).
	// After Close, behaviour of other methods is undefined.
	Close() error
}

// NoopNodeStorage satisfies NodeStorage with in-memory no-ops.
// It is the default when no persistent backend is configured, allowing the
// graph to operate purely in memory without any disk I/O.
type NoopNodeStorage struct{}

var _ NodeStorage = (*NoopNodeStorage)(nil)

// SaveNode is a no-op.
func (n *NoopNodeStorage) SaveNode(_, _ string, _ *Module) error { return nil }

// LoadNode always reports that the node does not exist.
func (n *NoopNodeStorage) LoadNode(_, _ string) (*Module, error) { return nil, nil }

// QueryEdges always returns an empty edge list.
func (n *NoopNodeStorage) QueryEdges(_, _ string) ([]string, error) { return nil, nil }

// Close is a no-op.
func (n *NoopNodeStorage) Close() error { return nil }
