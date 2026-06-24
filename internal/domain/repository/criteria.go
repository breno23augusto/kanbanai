package repository

type Operator string

const (
	OpEquals              Operator = "="
	OpNotEquals           Operator = "!="
	OpGreaterThan         Operator = ">"
	OpLessThan            Operator = "<"
	OpGreaterThanOrEquals Operator = ">="
	OpLessThanOrEquals    Operator = "<="
	OpLike                Operator = "LIKE"
	OpIn                  Operator = "IN"
)

type Criterion struct {
	Key      string
	Value    any
	Operator Operator
}

type Criteria []Criterion
