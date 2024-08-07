package brain

// Brain is a combined [Learner] and [Speaker].
type Brain interface {
	Learner
	Speaker
}
