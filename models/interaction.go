package models

type Like struct {
	ID        int64
	UserID    int64
	VideoID   int64
	IsDeleted int // 0:有效 1:删除
}

func (Like) TableName() string {
	return "likes"
}
