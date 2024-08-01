package updateuser

type Request struct {
	FirstName *string
	LastName  *string
	Nickname  *string
	Email     *string
	Password  *string
	Country   *string
}
