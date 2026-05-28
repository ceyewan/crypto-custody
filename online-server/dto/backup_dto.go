package dto

type ColdBackupRequest struct {
	Password string `json:"password" binding:"required"`
}

type RestoreBackupRequest struct {
	Password string `json:"password"`
}
