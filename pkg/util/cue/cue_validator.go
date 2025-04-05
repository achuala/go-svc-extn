package cue

type CueValidator struct {
}

func NewCueValidator(schemaDirectory string) (*CueValidator, error) {

	return &CueValidator{}, nil
}
