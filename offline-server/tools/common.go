package tools

// 定义用户的 Role 类型
type Role string

const (
	Admin Role = "admin"
	User  Role = "user"
	Guest Role = "guest"
)
