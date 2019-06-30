package events

type subSettings struct {
	buffer int
}

var subSettingsDefault = subSettings{
	buffer: 16,
}

func BufSize(n int) func(interface{}) error {
	return func(s interface{}) error {
		s.(*subSettings).buffer = n
		return nil
	}
}
