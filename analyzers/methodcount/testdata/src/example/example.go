package example

type OK struct{}

func (o OK) A() {}
func (o OK) B() {}

type TooMany struct{} // exported methods defined below

func (t TooMany) One() {}   // want `\[methodcount\] type TooMany has 3 exported methods \(max 2\)`
func (t TooMany) Two() {}
func (t TooMany) Three() {}
func (t TooMany) private() {} // not counted
