package codegen

// Tree node kind. It can be an array, map or a leaf.
type nodeKind int

const (
	arrayNode nodeKind = iota
	mapNode
	leaf
)

// Represents a node within a JSON structure. It is used in traversing
// the structure by the code generator.
type node struct {
	parent *node
	indent int
	kind   nodeKind
	key    string
}

// Instantiates a new root node of the given kind. The indent is the
// indentation of the root node. Child nodes derive from this value.
func newNode(indent int, kind nodeKind) *node {
	return &node{
		indent: indent,
		kind:   kind,
	}
}

// Checks if the specified node is a root.
func (n *node) isRoot() bool {
	return n.parent == nil
}

// Checks if the node's parent is an array.
func (n *node) isParentArray() bool {
	if n.isRoot() {
		return false
	}
	return n.parent.kind == arrayNode
}

// Checks if the node's parent is a map.
func (n *node) isParentMap() bool {
	if n.isRoot() {
		return false
	}
	return n.parent.kind == mapNode
}

// Returns node's indentation.
func (n *node) getIndentation() int {
	return n.indent
}

// Creates child node of the given kind.
func (n *node) createChild(kind nodeKind) *node {
	child := newNode(n.indent+1, kind)
	child.parent = n
	return child
}

func (n *node) createMapChild(key string, kind nodeKind) *node {
	child := n.createChild(kind)
	child.key = key
	return child
}
