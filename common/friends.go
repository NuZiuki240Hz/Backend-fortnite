package common

import (
	"fmt"
	"time"

	"github.com/zombman/server/all"
	"github.com/zombman/server/models"
)

type friendListEntry struct {
	AccountId string `json:"accountId"`
	Status    string `json:"status"`
	Direction string `json:"direction"`
	Created   string `json:"created"`
	Favorite  bool   `json:"favorite"`
}

func GetAllAcceptedFriends(accountId string) []friendListEntry {
	var friendList []friendListEntry

	var friendActions []models.FriendAction
	all.Postgres.Find(&friendActions, "for_account_id = ? AND action = ?", accountId, "ACCEPTED")

	var friendActions2 []models.FriendAction
	all.Postgres.Find(&friendActions2, "account_id = ? AND action = ?", accountId, "ACCEPTED")

	for _, friendAction := range friendActions {
		AddInFriendToResponse(accountId, friendAction, &friendList)
	}

	for _, friendAction := range friendActions2 {
		AddOutFriendToResponse(accountId, friendAction, &friendList)
	}

	return friendList
}

func GetFriendsList(accountId string) []friendListEntry {
	var friendList []friendListEntry

	var friendActions []models.FriendAction
	all.Postgres.Find(&friendActions, "for_account_id = ?", accountId)

	var friendActions2 []models.FriendAction
	all.Postgres.Find(&friendActions2, "account_id = ?", accountId)
	
	for _, friendAction := range friendActions {
		AddInFriendToResponse(accountId, friendAction, &friendList)
	}

	for _, friendAction := range friendActions2 {
		AddOutFriendToResponse(accountId, friendAction, &friendList)
	}

	return friendList
}

func GetPendingFriendsList(accountId string) []friendListEntry {
	var friendList []friendListEntry

	var friendActions []models.FriendAction
	all.Postgres.Find(&friendActions, "for_account_id = ? AND action = ?", accountId, "PENDING")

	var friendActions2 []models.FriendAction
	all.Postgres.Find(&friendActions2, "account_id = ? AND action = ?", accountId, "PENDING")

	for _, friendAction := range friendActions {
		AddInFriendToResponse(accountId, friendAction, &friendList)
	}

	for _, friendAction := range friendActions2 {
		AddOutFriendToResponse(accountId, friendAction, &friendList)
	}

	return friendList
}

func AddInFriendToResponse(accountId string,friendAction models.FriendAction, res *[]friendListEntry) {
	status := "PENDING"
	direction := "INBOUND"

	if friendAction.Action == "BLOCKED" {
		return
	}

	if friendAction.AccountId == accountId {
		direction = "OUTBOUND"
	}

	if friendAction.Action == "ACCEPTED" {
		status = "ACCEPTED"
		direction = "INBOUND"
	}

	*res = append(*res, friendListEntry{
		AccountId: friendAction.AccountId,
		Status:    status,
		Direction: direction,
		Created:   time.Now().AddDate(0, 0, -7).Format("2006-01-02T15:04:05.999Z"),
		Favorite:  false,
	})
}

func AddOutFriendToResponse(accountId string,friendAction models.FriendAction, res *[]friendListEntry) {
	status := "PENDING"
	direction := "INBOUND"

	if friendAction.Action == "BLOCKED" {
		return
	}

	if friendAction.AccountId == accountId {
		direction = "OUTBOUND"
	}

	if friendAction.Action == "ACCEPTED" {
		status = "ACCEPTED"
		direction = "INBOUND"
	}

	*res = append(*res, friendListEntry{
		AccountId: friendAction.ForAccountId,
		Status:    status,
		Direction: direction,
		Created:   time.Now().AddDate(0, 0, -7).Format("2006-01-02T15:04:05.999Z"),
		Favorite:  false,
	})
}

func IsFriend(accountId string, friendId string) bool {
	var friendActions []models.FriendAction
	all.Postgres.Find(&friendActions, "for_account_id = ? AND account_id = ? AND action = ?", accountId, friendId, "ACCEPTED")

	var friendActions2 []models.FriendAction
	all.Postgres.Find(&friendActions2, "for_account_id = ? AND account_id = ? AND action = ?", friendId, accountId, "ACCEPTED")

	return len(friendActions) > 0 || len(friendActions2) > 0
}

func IsBlocked(accountId string, friendId string) bool {
	var friendActions []models.FriendAction
	all.Postgres.Find(&friendActions, "for_account_id = ? AND account_id = ? AND action = ?", accountId, friendId, "BLOCKED")

	var friendActions2 []models.FriendAction
	all.Postgres.Find(&friendActions2, "for_account_id = ? AND account_id = ? AND action = ?", friendId, accountId, "BLOCKED")

	return len(friendActions) > 0 || len(friendActions2) > 0
}

func IsPending(accountId string, friendId string) bool {
	var friendActions []models.FriendAction
	all.Postgres.Find(&friendActions, "for_account_id = ? AND account_id = ? AND action = ?", friendId, accountId, "PENDING")
	return len(friendActions) > 0
}

func AcceptFriend(accountId string, friendId string) error {
	var pendingRequest models.FriendAction
	res := all.Postgres.Find(&pendingRequest, "for_account_id = ? AND account_id = ? AND action = ?", accountId, friendId, "PENDING")

	if res.RowsAffected <= 0 {
		return fmt.Errorf("found %d pending friend request from %s to %s", res.RowsAffected, friendId, accountId)
	}

	pendingRequest.Action = "ACCEPTED"
	all.Postgres.Save(&pendingRequest)

	return nil
}

func CreateFriend(accountId string, friendId string) string {//(models.FriendAction, error) {
	var pendingRequest models.FriendAction
	res := all.Postgres.Find(&pendingRequest, "for_account_id = ? AND account_id = ? AND action = ?", accountId, friendId, "PENDING")
	
	if res.RowsAffected > 0 {
		AcceptFriend(accountId, friendId)
		return "ACCEPTED"
	}

	friendAction := models.FriendAction{
		ForAccountId: friendId,
		AccountId:    accountId,
		Action:       "PENDING",
	}
	all.Postgres.Create(&friendAction)

	return "PENDING"
}

func DeleteFriend(accountId string, friendId string) string {
	var friendAction models.FriendAction
	res := all.Postgres.Find(&friendAction, "for_account_id = ? AND account_id = ?", accountId, friendId)
	if res.RowsAffected <= 0 {
		res2 := all.Postgres.Find(&friendAction, "for_account_id = ? AND account_id = ?", friendId, accountId)
		if res2.RowsAffected <= 0 {
			return ""
		}
		all.Postgres.Delete(&friendAction)
		return "DELETED"
	}

	all.Postgres.Delete(&friendAction)

	return "DELETED"
}

func GetBlockedFriendsList(accountId string) []friendListEntry {
	var blockedFriendActions []models.FriendAction
	all.Postgres.Find(&blockedFriendActions, "for_account_id = ? AND action = ?", accountId, "BLOCKED")

	var blockedFriends []friendListEntry
	for _, blockedFriendAction := range blockedFriendActions {
		blockedFriends = append(blockedFriends, friendListEntry{
			AccountId: blockedFriendAction.AccountId,
			Status:    "BLOCKED",
			Direction: "OUTBOUND",
			Created:   time.Now().Format("2006-01-02T15:04:05.999Z"),
			Favorite:  false,
		})
	}

	return blockedFriends
}

func BlockFriend(accountId string, friendId string) string {
	var friendData models.FriendAction
	meToFriendRes := all.Postgres.Find(&friendData, "for_account_id = ? AND account_id = ? AND action = ?", accountId, friendId, "ACCEPTED")

	if meToFriendRes.RowsAffected > 0 && friendData.Action == "ACCEPTED" {
		friendData.Action = "BLOCKED"
		all.Postgres.Save(&friendData)

		return "BLOCKED"
	}

	friendToMeRes := all.Postgres.Find(&friendData, "for_account_id = ? AND account_id = ? AND action = ?", friendId, accountId, "ACCEPTED")

	if friendToMeRes.RowsAffected > 0 && friendData.Action == "ACCEPTED" {
		friendData.Action = "BLOCKED"
		all.Postgres.Save(&friendData)
		return "BLOCKED"
	}

	return ""
}

func UnBlockFriend(accountId string, friendId string) {
	DeleteFriend(accountId, friendId)
}