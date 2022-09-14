package models

type ShortLink struct {
	ID        string `json:"id" dynamodbav:"id"`
	LinkURL   string `json:"linkUrl" dynamodbav:"linkUrl"`
	CreatedAt string `json:"createdAt" dynamodbav:"createdAt"`
	ExpiresAt string `json:"expiresAt,omitempty" dynamodbav:"expiresAt,omitempty"`
}
