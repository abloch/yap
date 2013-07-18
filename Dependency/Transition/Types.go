package Transition

type HasAttributes interface {
	GetProperty(property string) (string, bool)
}

type Token string

type Sentence []Token

type Stack interface {
	Push(int)
	Pop() (int, bool)
	Index(int) (int, bool)
	Peek() (int, bool)
	Size() int

	Copy() Stack
}

type Queue interface {
	Enqueue(int)
	Dequeue() (int, bool)
	Index(int) (int, bool)
	Peek() (int, bool)
	Size() int

	Copy() Queue
}

type ArcSet interface {
	Add(DepArc)
	Get(DepArc) []*DepArc
	Copy() ArcSet
}

type Configuration interface {
	HasAttributes

	Init(Sentence)
	Terminal() bool

	Stack() Stack
	Queue() Queue
	Arcs() ArcSet

	Copy() *Configuration
	GetSequence() []Configuration
	SetLastTransition(string)
}