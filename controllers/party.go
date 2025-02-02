package controllers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zombman/server/all"
	"github.com/zombman/server/common"
	"github.com/zombman/server/models"
	"github.com/zombman/server/socket"
)

func PartyGetUser(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	all.PrintMagenta([]any{"PartyGetUser"})
	
	partyId := common.AccountIdToPartyId[user.AccountId]
	party, ok := common.ActiveParties[partyId]
	if !ok {
		party = common.CreateParty(&common.ActiveParties, &common.AccountIdToPartyId, user)
	}
	
	c.JSON(200, gin.H{
		"current": []models.V2Party{party},
		"pending": []gin.H{},
		"invites": []gin.H{},
		"pings": []gin.H{},
	})
}

func PartyGetFriendPartyPings(c *gin.Context) {
	friend, err := common.GetUserByAccountId(c.Param("friendId"))
	if err != nil {
		common.ErrorBadRequest(c)
		return
	}

	partyId, ok := common.AccountIdToPartyId[friend.AccountId]
	if !ok {
		common.ErrorBadRequest(c)
		return
	}

	party, ok := common.ActiveParties[partyId]
	if !ok {
		common.ErrorBadRequest(c)
		return
	}

	c.JSON(200, []models.V2Party{party})
}

func PartyPost(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	var body struct {
		Config struct {
			JoinConfirmation bool `json:"join_confirmation"`
			Joinability string `json:"joinability"`
			MaxSize int `json:"max_size"`
		} `json:"config"`
		JoinInfo struct {
			Connection struct {
				Id string `json:"id"`
				Meta struct {
					UrnEpicConnPlatformS string `json:"urn:epic:conn:platform_s"`
					UrnEpicConnTypeS string `json:"urn:epic:conn:type_s"`
				} `json:"meta"`
			} `json:"connection"`
			Meta struct {
				UrnEpicMemberDnS string `json:"urn:epic:member:dn_s"`
				UrnEpicMemberTypeS string `json:"urn:epic:member:type_s"`
				UrnEpicMemberPlatformS string `json:"urn:epic:member:platform_s"`
			} `json:"meta"`
		} `json:"join_info"`
		Meta map[string]interface{} `json:"meta"`
	}

	if err := c.BindJSON(&body); err != nil {
		common.ErrorBadRequest(c)
		return
	}

	party := common.CreateParty(&common.ActiveParties, &common.AccountIdToPartyId, user)
	common.ActiveParties[party.ID] = party
	common.AccountIdToPartyId[user.AccountId] = party.ID

	connectionMeta := make(map[string]interface{})
	connectionMeta["urn:epic:conn:platform_s"] = body.JoinInfo.Connection.Meta.UrnEpicConnPlatformS
	connectionMeta["urn:epic:conn:type_s"] = body.JoinInfo.Connection.Meta.UrnEpicConnTypeS
	connection := models.V2PartyConnection{
		ID: body.JoinInfo.Connection.Id,
		Meta: connectionMeta,
		YieldLeadership: false,
		ConnectedAt: time.Now().Format("2006-01-02T15:04:05.999Z"),
		UpdatedAt: time.Now().Format("2006-01-02T15:04:05.999Z"),
	}

	partyMemberMeta := make(map[string]interface{})
	partyMemberMeta["urn:epic:member:dn_s"] = user.Username
	partyMemberMeta["urn:epic:member:joinrequestusers_j"] = "{\"users\":[{\"id\":\""+ user.AccountId +"\",\"dn\":\""+ user.Username +"\",\"plat\":\"WIN\",\"data\":\"{\\\"CrossplayPreference_i\\\":\\\"1\\\",\\\"SubGame_u\\\":\\\"1\\\"}\"}]}"
	partyMember := models.V2PartyMember{
		AccountId: user.AccountId,
		Meta: partyMemberMeta,
		Connections: []models.V2PartyConnection{connection},
		Role: "CAPTAIN",
		Revision: 0,
		JoinedAt: time.Now().Format("2006-01-02T15:04:05.999Z"),
		UpdatedAt: time.Now().Format("2006-01-02T15:04:05.999Z"),
	}

	party.Config.JoinConfirmation = true
	party.Config.Joinability = "OPEN"
	party.Config.MaxSize = 4
	party.Members = []models.V2PartyMember{partyMember}

	for key, metaItem := range body.Meta {
		party.Meta[key] = metaItem
	}

	common.ActiveParties[party.ID] = party

	memberClient, _ := socket.XGetClientFromAccountId(user.AccountId)

	socket.XMPPSendBody(gin.H{
		"account_id": partyMember.AccountId,
		"account_dn": partyMember.Meta["urn:epic:member:dn_s"],
		"connection": gin.H{
			"id": partyMember.Connections[0].ID,
			"meta": partyMember.Connections[0].Meta,
			"updated_at": partyMember.Connections[0].UpdatedAt,
			"connected_at": partyMember.Connections[0].ConnectedAt,
			"joined_at": time.Now().Format("2006-01-02T15:04:05.000Z"),
		},
		"member_state_updated": partyMember.Meta,
		"party_id": party.ID,
		"updated_at": partyMember.UpdatedAt,
		"joined_at": partyMember.JoinedAt,
		"sent": time.Now().Format("2006-01-02T15:04:05.000Z"),
		"revision": party.Revision,
		"ns": "Fortnite",
		"type": "com.epicgames.social.party.notification.v0.MEMBER_JOINED",
	}, memberClient)

	c.JSON(200, party)

	common.DeleteEmptyParties()
}

func PartyPatch(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	var body struct {
		Config struct {
			JoinConfirmation bool `json:"join_confirmation"`
			Joinability string `json:"joinability"`
			MaxSize int `json:"max_size"`
		} `json:"config"`
		JoinInfo struct {
			Connection struct {
				Id string `json:"id"`
				Meta struct {
					UrnEpicConnPlatformS string `json:"urn:epic:conn:platform_s"`
					UrnEpicConnTypeS string `json:"urn:epic:conn:type_s"`
				} `json:"meta"`
			} `json:"connection"`
			Meta struct {
				UrnEpicMemberDnS string `json:"urn:epic:member:dn_s"`
				UrnEpicMemberTypeS string `json:"urn:epic:member:type_s"`
				UrnEpicMemberPlatformS string `json:"urn:epic:member:platform_s"`
			} `json:"meta"`
		} `json:"join_info"`
		Meta struct {
			Update map[string]interface{} `json:"update"`
			Delete []string `json:"delete"`
		} `json:"meta"`
	}

	if err := c.BindJSON(&body); err != nil {
		common.ErrorBadRequest(c)
		return
	}

	partyId := common.AccountIdToPartyId[user.AccountId]
	party, ok := common.ActiveParties[partyId]
	if !ok {
		common.ErrorBadRequest(c)
		return
	}

	for _, key := range body.Meta.Delete {
		delete(party.Meta, key)
		delete(party.Meta, strings.ReplaceAll(key, "Default:", ""))
	}

	for key, metaItem := range body.Meta.Update {
		party.Meta[key] = metaItem
		party.Meta[strings.ReplaceAll(key, "Default:", "")] = metaItem
	}

	party.Config.JoinConfirmation = true
	party.Config.Joinability = "OPEN"
	party.Config.MaxSize = 4
	common.ActiveParties[partyId] = party

	var captain models.V2PartyMember
	for _, member := range party.Members {
		if member.Role == "CAPTAIN" {
			captain = member
		}
	}

	for _, member := range party.Members {
		memberClient, err := socket.XGetClientFromAccountId(member.AccountId)
		if err != nil {
			continue
		}

		socket.XMPPSendBody(gin.H{
			"captain_id":            captain.AccountId,
			"created_at":            party.CreatedAt,
			"invite_ttl_seconds":    party.Config.InviteTtl,
			"max_number_of_members": 4,
			"ns":                    "Fortnite",
			"party_id":              party.ID,
			"party_privacy_type":    "OPEN",
			"party_state_overriden": gin.H{},
			"party_state_removed":   []string{},
			"party_state_updated":   party.Meta,
			"party_sub_type":        "default",
			"party_type":            "DEFAULT",
			"revision":              party.Revision,
			"sent":                  time.Now().Format("2006-01-02T15:04:05.999Z"),
			"type":                  "com.epicgames.social.party.notification.v0.PARTY_UPDATED",
			"updated_at":            time.Now().Format("2006-01-02T15:04:05.999Z"),
		}, memberClient)
	}

	c.JSON(200, party)
}

func PartyPatchMemberMeta(c *gin.Context) {
	partyId := c.Param("partyId")
	memberId := c.Param("memberId")

	party, ok := common.ActiveParties[partyId]
	if !ok {
		common.ErrorBadRequest(c)
		return
	}

	var body struct {
		Update map[string]interface{} `json:"update"`
		Delete []string `json:"delete"`
	}

	if err := c.BindJSON(&body); err != nil {
		common.ErrorBadRequest(c)
		return
	}

	for _, member := range party.Members {
		if member.AccountId == memberId {
			for _, key := range body.Delete {
				delete(member.Meta, key)
				delete(member.Meta, strings.ReplaceAll(key, "Default:", ""))
			}

			for key, metaItem := range body.Update {
				member.Meta[key] = metaItem
				member.Meta[strings.ReplaceAll(key, "Default:", "")] = metaItem
			}

			break
		}
	}

	var partyMember models.V2PartyMember
	for _, member := range party.Members {
		if member.AccountId == memberId {
			partyMember = member
			break
		}
	}
	common.ActiveParties[partyId] = party

	for _, member := range party.Members {
		memberClient, err := socket.XGetClientFromAccountId(member.AccountId)
		if err != nil {
			continue
		}

		socket.XMPPSendBody(gin.H{
			"account_id": partyMember.AccountId,
			"account_dn": partyMember.Meta["urn:epic:member:dn_s"],
			"member_state_updated": partyMember.Meta,
			"member_state_removed": []string{},
			"member_state_overridden": gin.H{},
			"party_id": party.ID,
			"updated_at": partyMember.UpdatedAt,
			"sent": time.Now().Format("2006-01-02T15:04:05.000Z"),
			"revision": party.Revision,
			"ns": "Fortnite",
			"type": "com.epicgames.social.party.notification.v0.MEMBER_STATE_UPDATED",
		}, memberClient)
	}

	c.JSON(200, party)
}

func PartyJoinMember(c *gin.Context) {
	user := c.MustGet("user").(models.User)
	partyId := c.Param("partyId")
	
	party, ok := common.ActiveParties[partyId]
	if !ok {
		common.ErrorBadRequest(c)
		return
	}

	common.LeaveOldParty(user.AccountId)

	var body struct {
		Connection struct {
			Id string `json:"id"`
			Meta struct {
				UrnEpicConnPlatformS string `json:"urn:epic:conn:platform_s"`
			} `json:"meta"`
		} `json:"connection"`
		Meta struct {
			UrnEpicMemberDnS string `json:"urn:epic:member:dn_s"`
			UrnJoinRequestUsers string `json:"urn:epic:member:joinrequestusers_j"`
		} `json:"meta"`
	}

	if err := c.BindJSON(&body); err != nil {
		common.ErrorBadRequest(c)
		return
	}

	var captain models.V2PartyMember
	for _, member := range party.Members {
		if member.Role == "CAPTAIN" {
			captain = member
			break
		}
	}

	connectionMeta := make(map[string]interface{})
	connectionMeta["urn:epic:conn:platform_s"] = body.Connection.Meta.UrnEpicConnPlatformS
	connection := models.V2PartyConnection{
		ID: body.Connection.Id,
		Meta: connectionMeta,
		YieldLeadership: false,
		ConnectedAt: time.Now().Format("2006-01-02T15:04:05.999Z"),
		UpdatedAt: time.Now().Format("2006-01-02T15:04:05.999Z"),
	}

	partyMemberMeta := make(map[string]interface{})
	partyMemberMeta["urn:epic:member:dn_s"] = body.Meta.UrnEpicMemberDnS
	partyMemberMeta["urn:epic:member:joinrequestusers_j"] = body.Meta.UrnJoinRequestUsers

	partyMember := models.V2PartyMember{
		AccountId: user.AccountId,
		Meta: partyMemberMeta,
		Connections: []models.V2PartyConnection{connection},
		Role: "MEMBER",
		Revision: 0,
		JoinedAt: time.Now().Format("2006-01-02T15:04:05.999Z"),
		UpdatedAt: time.Now().Format("2006-01-02T15:04:05.999Z"),
	}

	party.Members = append(party.Members, partyMember)
	
	captianJoinRequests := models.V2CaptainJoinRequestUsers{
		Users: []models.V2CaptainJoinRequestUser{},
	}
	rawSquadAssignments := models.V2RawSquadAssignments{
		RawSquadAssignments: []models.V2RawSquadAssignment{},
	}

	for i, member := range party.Members {
		captianJoinRequests.Users = append(captianJoinRequests.Users, models.V2CaptainJoinRequestUser{
			ID: member.AccountId,
			DisplayName: member.Meta["urn:epic:member:dn_s"].(string),
			Platform: "WIN",
			Data: "{\"CrossplayPreference_i\":\"1\"}",
		})
		rawSquadAssignments.RawSquadAssignments = append(rawSquadAssignments.RawSquadAssignments, models.V2RawSquadAssignment{
			MemberId: member.AccountId,
			AbsoluteMemberIdx: i,
		})
	}

	for i, member := range party.Members {
		if member.Role == "CAPTAIN" {
			party.Members = append(party.Members[:i], party.Members[i+1:]...)
			break
		}
	}

	captianJoinRequestsRaw, _ := json.Marshal(captianJoinRequests)
	captain.Meta["urn:epic:member:joinrequestusers_j"] = string(captianJoinRequestsRaw)

	rawSquadAssignmentsRaw, _ := json.Marshal(rawSquadAssignments)
	party.Meta["RawSquadAssignments_j"] = string(rawSquadAssignmentsRaw)

	party.Members = append(party.Members, captain)
	common.ActiveParties[partyId] = party
	common.AccountIdToPartyId[user.AccountId] = partyId

	for _, member := range party.Members {
		memberClient, err := socket.XGetClientFromAccountId(member.AccountId)
		if err != nil {
			continue
		}

		socket.XMPPSendBody(gin.H{
			"account_id": partyMember.AccountId,
			"account_dn": partyMember.Meta["urn:epic:member:dn_s"],
			"connection": gin.H{
				"id": partyMember.Connections[0].ID,
				"meta": partyMember.Connections[0].Meta,
				"updated_at": partyMember.Connections[0].UpdatedAt,
				"connected_at": partyMember.Connections[0].ConnectedAt,
			},
			"member_state_updated": partyMember.Meta,
			"party_id": party.ID,
			"updated_at": partyMember.UpdatedAt,
			"joined_at": partyMember.JoinedAt,
			"sent": time.Now().Format("2006-01-02T15:04:05.000Z"),
			"revision": party.Revision,
			"ns": "Fortnite",
			"type": "com.epicgames.social.party.notification.v0.MEMBER_JOINED",
		}, memberClient)
	}
	
	c.JSON(201, gin.H{
		"status": "JOINED",
		"party_id": partyId,
	})
}

func PartyGet(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	partyId := c.Param("partyId")
	party, ok := common.ActiveParties[partyId]
	if !ok {
		common.ErrorBadRequest(c)
		return
	}

	all.PrintMagenta([]any{
		"PartyGet",
		partyId,
		user.Username,
	})

	c.JSON(200, party)
}

func PartyDeleteMember(c *gin.Context) {
	partyId := c.Param("partyId")
	memberId := c.Param("memberId")

	party, ok := common.ActiveParties[partyId]
	if !ok {
		common.ErrorBadRequest(c)
		return
	}

	var removingMember models.V2PartyMember
	for i, member := range party.Members {
		if member.AccountId == memberId {
			removingMember = member
			party.Members = append(party.Members[:i], party.Members[i+1:]...)
			break
		}
	}

	if len(party.Members) == 0 {
		delete(common.ActiveParties, partyId)
	}

	if removingMember.Role == "CAPTAIN" {
		if len(party.Members) == 0 {
			delete(common.ActiveParties, partyId)
			return
		}

		party.Members[0].Role = "CAPTAIN"
	}

	delete(common.AccountIdToPartyId, memberId)
	common.ActiveParties[partyId] = party

	for _, member := range party.Members {
		memberClient, err := socket.XGetClientFromAccountId(member.AccountId)
		if err != nil {
			continue
		}

		socket.XMPPSendBody(gin.H{
			"account_id": removingMember.AccountId,
			"party_id": party.ID,
			"sent": time.Now().Format("2006-01-02T15:04:05.000Z"),
			"revision": party.Revision,
			"ns": "Fortnite",
			"type": "com.epicgames.social.party.notification.v0.MEMBER_LEFT",
		}, memberClient)
	}

	c.JSON(200, party)

	common.DeleteEmptyParties()
}