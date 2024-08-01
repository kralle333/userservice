package listusers

type PageInfo struct {
	Limit  int64  `json:"limit"`
	Cursor string `json:"cursor"`
}

type SortInfo struct {
	By    UserField `json:"by"`
	Order *Ordering `json:"order,omitempty"`
}

type FilterInfo struct {
	Left     UserField `json:"left"`
	Comparer Comparer  `json:"comparer"`
	Right    string    `json:"right"`
}

type Request struct {
	Sorting   *SortInfo   `json:"sorting,omitempty"`
	Filtering *FilterInfo `json:"filtering,omitempty"`
	Paging    *PageInfo   `json:"paging,omitempty"`
}
