package types

import "time"

// User represents a user in the system
type User struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name" validate:"required"`
	Email     string    `json:"email" db:"email" validate:"required,email"`
	Age       int       `json:"age" db:"age" validate:"min=0,max=150"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreateUserRequest represents the request payload for creating a user
type CreateUserRequest struct {
	Name  string `json:"name" minLength:"1" maxLength:"100" example:"John Doe" doc:"User's full name"`
	Email string `json:"email" format:"email" example:"john@example.com" doc:"User's email address"`
	Age   int    `json:"age" minimum:"0" maximum:"150" example:"30" doc:"User's age"`
}

// UpdateUserRequest represents the request payload for updating a user
type UpdateUserRequest struct {
	Name  string `json:"name,omitempty" minLength:"1" maxLength:"100" example:"John Doe" doc:"User's full name"`
	Email string `json:"email,omitempty" format:"email" example:"john@example.com" doc:"User's email address"`
	Age   int    `json:"age,omitempty" minimum:"0" maximum:"150" example:"30" doc:"User's age"`
}

// Post represents a blog post or article
type Post struct {
	ID        int       `json:"id" db:"id"`
	Title     string    `json:"title" db:"title" validate:"required,min=1,max=200"`
	Content   string    `json:"content" db:"content" validate:"required,min=10"`
	UserID    int       `json:"user_id" db:"user_id" validate:"required"`
	User      *User     `json:"user,omitempty" db:"-"`
	Published bool      `json:"published" db:"published"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreatePostRequest represents the request payload for creating a post
type CreatePostRequest struct {
	Title     string `json:"title" minLength:"1" maxLength:"200" example:"My First Post" doc:"Post title"`
	Content   string `json:"content" minLength:"10" example:"This is the content of my post..." doc:"Post content"`
	UserID    int    `json:"user_id" minimum:"1" example:"1" doc:"ID of the user creating the post"`
	Published bool   `json:"published" example:"false" doc:"Whether the post is published"`
}

// UpdatePostRequest represents the request payload for updating a post
type UpdatePostRequest struct {
	Title     string `json:"title,omitempty" minLength:"1" maxLength:"200" example:"Updated Post Title" doc:"Post title"`
	Content   string `json:"content,omitempty" minLength:"10" example:"Updated content..." doc:"Post content"`
	UserID    int    `json:"user_id,omitempty" minimum:"1" example:"1" doc:"ID of the user who owns the post"`
	Published bool   `json:"published,omitempty" example:"true" doc:"Whether the post is published"`
}

// Comment represents a comment on a post
type Comment struct {
	ID        int       `json:"id" db:"id"`
	Content   string    `json:"content" db:"content" validate:"required,min=1,max=500"`
	PostID    int       `json:"post_id" db:"post_id" validate:"required"`
	UserID    int       `json:"user_id" db:"user_id" validate:"required"`
	Post      *Post     `json:"post,omitempty" db:"-"`
	User      *User     `json:"user,omitempty" db:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Category represents a content category
type Category struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name" validate:"required,min=1,max=100"`
	Description string `json:"description" db:"description"`
	Color       string `json:"color" db:"color" validate:"hexcolor"`
}

// Tag represents a content tag
type Tag struct {
	ID   int    `json:"id" db:"id"`
	Name string `json:"name" db:"name" validate:"required,min=1,max=50"`
	Slug string `json:"slug" db:"slug" validate:"required"`
}

// PostCategory represents the many-to-many relationship between posts and categories
type PostCategory struct {
	PostID     int `json:"post_id" db:"post_id"`
	CategoryID int `json:"category_id" db:"category_id"`
}

// PostTag represents the many-to-many relationship between posts and tags
type PostTag struct {
	PostID int `json:"post_id" db:"post_id"`
	TagID  int `json:"tag_id" db:"tag_id"`
}

// UserProfile extends user information
type UserProfile struct {
	UserID    int     `json:"user_id" db:"user_id"`
	Bio       *string `json:"bio,omitempty" db:"bio"`
	Avatar    *string `json:"avatar,omitempty" db:"avatar"`
	Website   *string `json:"website,omitempty" db:"website" validate:"omitempty,url"`
	Location  *string `json:"location,omitempty" db:"location"`
	BirthDate *string `json:"birth_date,omitempty" db:"birth_date"`
}

// APIResponse is a generic response wrapper
type APIResponse[T any] struct {
	Data    T      `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Status represents various status types
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusPending  Status = "pending"
	StatusArchived Status = "archived"
)

// Role represents user roles
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleModerator Role = "moderator"
	RoleUser      Role = "user"
	RoleGuest     Role = "guest"
)

// Priority levels
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
) 