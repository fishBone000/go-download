package main

type Node[T any] struct {
	prev    *Node[T]
	next    *Node[T]
	content T
}

func (node Node[T]) Prev() *Node[T] {
	return node.prev
}

func (node Node[T]) Next() *Node[T] {
	return node.next
}

func (node Node[T]) Content() T {
	return node.content
}

func (node Node[T]) Set(e T) {
	node.content = e
}

func (node Node[T]) Del() {
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
}

type List[T any] struct {
	head *Node[T]
	tail *Node[T]
	zero T
}

func (list *List[T]) At(n int) (T, bool) {
	if list.head == nil || n < 0 {
		return list.zero, false
	}

	p := list.head
	for i := 0; i != n; i++ {
		if p.next == nil {
			return list.zero, false
		}
		p = p.next
	}
	return p.content, true
}

func (list *List[T]) Set(n int, e T) bool {
	if list.head == nil || n < 0 {
		return false
	}

	p := list.head
	for i := 0; i != n; i++ {
		if p.next == nil {
			return false
		}
		p = p.next
	}
	p.content = e
	return true
}

func (list *List[T]) Insert(n int, e T) (*Node[T], bool) {
	if list.head == nil || n < 0 {
		return nil, false
	}

	i := 0
	p := list.head
	for i != n {
		if i == n-1 && p == list.tail {
			p.next = &Node[T]{
				prev: 		p,
				next: 		nil,
				content: 	e,
			}
			list.tail = p.next
			return list.tail, true
		}
		if p.next == nil {
			return nil, false
		}
		p = p.next
		i++
	}
	node := Node[T]{
		prev:		p.prev,
		next:		p,
		content: 	e,
	}
	if p.prev != nil {
		p.prev.next = &node
	}
	p.prev = &node
	if p == list.head {
		list.head = p.prev
	}
	return &node, true
}

func (list *List[T]) InsNode(p *Node[T], e T) bool {
	if p == nil {
		return false
	}
	n := &Node[T]{
		next:		p,
		prev:		p.prev,
		content:	e,
	}
	if p == list.head {
		list.head = n
	}
	if p.prev != nil {
		p.prev.next = n
	}
	p.prev = n
	return true
}

func (list *List[T]) Del(n int) bool {
	if list.head == nil || n < 0 {
		return false
	}
	
	p := list.head
	for i := 0; i != n; i++ {
		if p.next == nil {
			return false
		}
		p = p.next
	}

	list.DelNode(p)
	return true
}

func (list *List[T]) DelNode(p *Node[T]) bool {
	if p == nil {
		return false
	}
	if p.prev != nil {
		p.prev.next = p.next
	}
	if p.next != nil {
		p.next.prev = p.prev
	}
	if p == list.head {
		list.head = p.next
	}
	if p == list.tail {
		list.tail = p.prev
	}
	return true
}

func (list List[T]) Append(e T) {
	if list.head == nil {
		list.head = &Node[T]{
			content: e,
		}
		list.tail = list.head
		return
	}

	p := list.tail
	p.next = &Node[T]{
		prev:		p,
		content:	e,
	}
	list.tail = p.next
}

func (list List[T]) Empty() bool {
	return list.head == nil
}

func (list List[T]) Front() *Node[T] {
	return list.head
}