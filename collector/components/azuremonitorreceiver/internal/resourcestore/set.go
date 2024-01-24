package resourcestore

type set map[string]struct{}

func newSet() set {
	return make(set)
}

func (s set) union(s2 set) set {
	s3 := make(set)
	for k := range s {
		s3[k] = struct{}{}
	}
	for k := range s2 {
		s3[k] = struct{}{}
	}
	return s3
}

func (s set) difference(s2 set) set {
	s3 := make(set)
	for k := range s {
		if _, ok := s2[k]; !ok {
			s3[k] = struct{}{}
		}
	}
	return s3
}

func (s set) intersection(s2 set) set {
	s3 := make(set)
	for k := range s {
		if _, ok := s2[k]; ok {
			s3[k] = struct{}{}
		}
	}
	return s3
}

func (s set) contains(s2 string) bool {
	_, ok := s[s2]
	return ok
}
