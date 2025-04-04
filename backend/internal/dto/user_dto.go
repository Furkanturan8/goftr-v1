package dto

import (
	"goftr-v1/backend/internal/model"
)

type CreateUserRequest struct {
	Email     string       `json:"email" validate:"required_without=Phone,omitempty,max=64,email"`
	FirstName string       `json:"first_name" validate:"required,max=100"`
	LastName  string       `json:"last_name" validate:"required,max=100"`
	Password  string       `json:"password" validate:"required,min=3,max=100"`
	Status    model.Status `json:"status" validate:"omitempty,oneof=active inactive"`
	Role      model.Role   `json:"role" validate:"required"`
}

func (dto CreateUserRequest) ToDBModel(m model.User) model.User {
	m.Email = dto.Email
	m.FirstName = dto.FirstName
	m.LastName = dto.LastName
	if dto.Password != "" {
		_ = m.SetPassword(dto.Password)
	}
	m.Role = dto.Role
	if dto.Status == "" {
		m.Status = model.StatusActive
	} else {
		m.Status = dto.Status
	}

	return m
}

type UserResponse struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	Status    string `json:"status"`
}

func (dto UserResponse) ToResponseModel(m model.User) UserResponse {
	dto.ID = m.ID
	dto.Email = m.Email
	dto.FirstName = m.FirstName
	dto.LastName = m.LastName
	dto.Role = string(m.Role)
	dto.Status = string(m.Status)

	return dto
}
