package util

type Int64Slice struct {
	Data []int64
}

func NewInt64Slice(data []int64) *Int64Slice {
	return &Int64Slice{
		Data:data,
	}
}


func (s *Int64Slice)Len() int {
	return len(s.Data)
}

func (s *Int64Slice)Less(i,j int) bool {
	return s.Data[i] <= s.Data[j]
}

func (s *Int64Slice)Swap(i,j int)  {
	s.Data[i],s.Data[j] = s.Data[j],s.Data[i]
}


