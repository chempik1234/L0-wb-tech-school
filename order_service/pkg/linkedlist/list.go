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

// ErrInvalidIndex describes an error when index is incorrect
//
// If length is 0 then use ErrEmptyList
var ErrInvalidIndex = errors.New("invalid index")

// ErrEmptyList describes an error when trying to get and element but there mustn't be any
//
// Because length = 0
var ErrEmptyList = errors.New("zero length")

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

	if index > currentLen {
		return ErrInvalidIndex
	}

	if index == 0 {
		newNode.next = l.head
		l.head = newNode
	} else if index > 0 {
		if index < currentLen {
			prev := l.head
			for i := 0; i < index-1; i++ {
				prev = prev.next
			}
			if prev.next != nil {
				newNode.next = prev.next
			}
			prev.next = newNode
		} else if index == currentLen { // and it's certainly > 0, so there's a tail
			l.tail.next = newNode
		}
	} else {
		return ErrInvalidIndex
	}
	l.length++

	if index == currentLen {
		l.setTail(newNode)
	}

	return nil
}

// InsertLast inserts a value after the last Node of the LinkedList
func (l *LinkedList[ValueType]) InsertLast(data ValueType) error {
	return l.Insert(data, l.Len())
}

//endregion

//region remove

// RemoveAt removes element by given index
func (l *LinkedList[ValueType]) RemoveAt(index int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.removeNodeAt(index)
}

// RemoveFirst removes first element
func (l *LinkedList[ValueType]) RemoveFirst() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.removeFirstNode()
}

// RemoveLast removes last element from the list
//
// Is tolerant to 0 length
func (l *LinkedList[ValueType]) RemoveLast() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.removeNodeAt(l.Len() - 1)
}

func (l *LinkedList[ValueType]) removeNodeAt(index int) error {
	length := l.Len()
	if length == 0 {
		return ErrEmptyList
	}

	if index == 0 {
		return l.removeFirstNode()
	}

	if index < 0 || index >= length {
		return ErrInvalidIndex
	}

	prevElem := l.head
	for i := 0; i < index-1; i++ {
		prevElem = prevElem.next
	}

	deletedElem := prevElem.next
	if deletedElem != nil {
		prevElem.next = deletedElem.next
	} else {
		prevElem.next = nil
	}

	if index == l.Len()-1 {
		l.setTail(prevElem)
	}
	l.length--
	return nil
}

func (l *LinkedList[ValueType]) removeFirstNode() error {
	length := l.Len()
	if length == 0 {
		return ErrEmptyList
	}

	l.head = l.head.next
	l.length--

	l.recalculateTail()

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
	if node == nil {
		return *new(ValueType), ErrEmptyList
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
	if node == nil {
		return *new(ValueType), ErrEmptyList
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
		return nil, ErrEmptyList
	}
	return l.head, nil
}

func (l *LinkedList[ValueType]) getNodeAt(index int) (*Node[ValueType], error) {
	length := l.Len()
	if length == 0 {
		return nil, ErrEmptyList
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
		return nil, ErrEmptyList
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

		nextNode := nodeToMove.next

		currentHead := l.head

		// time to write changes
		// even if the list is critically small, these don't contradict each other
		l.head = nodeToMove
		nodeToMove.next = currentHead
		prevNode.next = nextNode

		// find tail
		l.recalculateTail()
	}
	return nil
}

// GetIndex tries to find first entry of element
//
// return index -1 if not found
//
// required custom compareFunc because ValueType is not always comparable
//
// compareFunc must return true if values is equal, else false
func (l *LinkedList[ValueType]) GetIndex(
	item ValueType,
	compareFunc func(item1 ValueType, item2 ValueType) bool,
) (int, error) {
	if l.Len() == 0 {
		return -1, ErrEmptyList
	}
	elem := l.head
	for i := 0; i < l.Len(); i++ {
		if compareFunc(elem.value, item) {
			return i, nil
		}
		elem = elem.next
	}
	return -1, nil
}

// GetAll returns a slice of ValueType, in same order as stored
func (l *LinkedList[ValueType]) GetAll() []ValueType {
	result := make([]ValueType, l.Len())
	currentElem := l.head
	for i := 0; i < l.Len(); i++ {
		result[i] = currentElem.value
		currentElem = currentElem.next
	}
	return result
}

func (l *LinkedList[ValueType]) recalculateTail() {
	currentElem := l.head
	for i := 0; i < l.Len()-1; i++ {
		currentElem = currentElem.next
	}
	l.setTail(currentElem)
}

func (l *LinkedList[ValueType]) setTail(newTail *Node[ValueType]) {
	l.tail = newTail
}
