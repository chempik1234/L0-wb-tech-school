package linkedlist

import (
	"errors"
	"sync"
)

// Node is the generic LinkedList Node type, any ValueType supported
//
// Stores the link to the next Node to loop through Nodes
type Node[ValueType any] struct {
	value ValueType
	next  *Node[ValueType]
}

// LinkedList is a data structure that stores first and last element, each of them has a link on the next one
type LinkedList[ValueType any] struct {
	head   *Node[ValueType]
	tail   *Node[ValueType]
	length int
	mu     sync.Mutex
}

// ErrInvalidIndex describes an error when there is anything wrong with given index, e.g. length=0 or index<0
var ErrInvalidIndex = errors.New("invalid index")

// NewLinkedList creates a new LinkedList with given ValueType, any ValueType is supported
func NewLinkedList[ValueType any]() LinkedList[ValueType] {
	return LinkedList[ValueType]{
		head:   nil,
		tail:   nil,
		length: 0,
	}
}

//region insert

// Insert inserts a value at certain index with right shift
func (l *LinkedList[ValueType]) Insert(data ValueType, index int) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	newNode := &Node[ValueType]{data, nil}
	currentLen := l.Len()

	if index == 0 {
		if currentLen > 0 {
			newNode.next = l.head
		}
		l.head = newNode
	} else if index > 0 {
		prev := l.head
		for i := 0; i < index-1; i++ {
			prev = prev.next
		}
		if prev.next != nil {
			newNode.next = prev.next
		}
		prev.next = newNode
	} else {
		return ErrInvalidIndex
	}
	if currentLen == 0 {
		l.tail = newNode
	}
	l.length++

	return nil
}

// InsertLast inserts a value after the last Node of the LinkedList
func (l *LinkedList[ValueType]) InsertLast(data ValueType) error {
	return l.Insert(data, l.Len())
}

//endregion

//region remove

func (l *LinkedList[ValueType]) RemoveAt(index int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.removeNodeAt(index)
}

func (l *LinkedList[ValueType]) RemoveFirst() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.removeFirstNode()
}

func (l *LinkedList[ValueType]) RemoveLast() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.removeNodeAt(l.Len() - 1)
}

func (l *LinkedList[ValueType]) removeNodeAt(index int) error {
	if index == 0 {
		return l.removeFirstNode()
	}

	if index < 0 || index >= l.Len() {
		return ErrInvalidIndex
	}

	var prevElem *Node[ValueType]
	currentElem := l.head
	for currentElem.next != nil {
		prevElem = currentElem
		currentElem = currentElem.next
	}

	prevElem.next = nil
	l.length--
	return nil
}

func (l *LinkedList[ValueType]) removeFirstNode() error {
	if l.Len() > 0 {
		l.head = l.head.next
		l.length--
	} else {
		return ErrInvalidIndex
	}
	return nil
}

//endregion

//region get

func (l *LinkedList[ValueType]) GetAt(index int) (ValueType, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	node, err := l.getNodeAt(index)
	if err != nil {
		return *new(ValueType), err
	}
	return node.value, nil
}

func (l *LinkedList[ValueType]) GetFirst() (ValueType, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	node, err := l.getFirstNode()
	if err != nil {
		return *new(ValueType), err
	}
	return node.value, nil
}

func (l *LinkedList[ValueType]) GetLast() (ValueType, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	node, err := l.getLastNode()
	if err != nil {
		return *new(ValueType), err
	}
	return node.value, nil
}

func (l *LinkedList[ValueType]) getFirstNode() (*Node[ValueType], error) {
	if l.Len() == 0 {
		return nil, ErrInvalidIndex
	}
	return l.head, nil
}

func (l *LinkedList[ValueType]) getNodeAt(index int) (*Node[ValueType], error) {
	length := l.Len()
	if length == 0 {
		return nil, ErrInvalidIndex
	}

	if index == 0 {
		return l.getFirstNode()
	} else if index < 0 || index >= length {
		return nil, ErrInvalidIndex
	} else if index == length-1 {
		return l.getLastNode()
	}

	currentNode := l.head
	for i := 0; i < index; i++ {
		currentNode = currentNode.next
	}
	return currentNode, nil
}

func (l *LinkedList[ValueType]) getLastNode() (*Node[ValueType], error) {
	if l.Len() == 0 {
		return nil, ErrInvalidIndex
	}
	return l.tail, nil
}

func (l *LinkedList[_]) Len() int {
	return l.length
}

//endregion

// MoveToFirst finds the element and moves it to index 0
func (l *LinkedList[ValueType]) MoveToFirst(from int) error {
	length := l.Len()

	if from < 0 || from >= length {
		return ErrInvalidIndex
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// no need to move otherwise
	if from > 0 && length > 1 {
		prevNode, err := l.getNodeAt(from - 1)
		if err != nil {
			return err
		}

		// by this moment we're sure it exists
		nodeToMove := prevNode.next

		nextNode := prevNode.next

		currentHead := l.head

		// time to write changes
		// even if the list is critically small, these don't contradict each other
		l.head = nodeToMove
		nodeToMove.next = currentHead
		prevNode.next = nextNode
	}
	return nil
}
