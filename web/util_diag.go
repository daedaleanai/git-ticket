package web

type diagnostic struct {
	Severity string
	Message  string
}

func diagNote(m string) *diagnostic {
	return &diagnostic{
		Severity: "note",
		Message:  m,
	}
}

func diagWarning(m string) *diagnostic {
	return &diagnostic{
		Severity: "warning",
		Message:  m,
	}
}

func diagError(m string) *diagnostic {
	return &diagnostic{
		Severity: "error",
		Message:  m,
	}
}
