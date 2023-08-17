package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zombman/server/all"
	"github.com/zombman/server/common"
	"github.com/zombman/server/controllers"
	"github.com/zombman/server/middleware"
	"github.com/zombman/server/models"
	"github.com/zombman/server/socket"
)

func init() {
  all.LoadEnviroment()
  all.ConnectToDatabase()
  all.AutoMigrate()
  common.InitGameServers()
  socket.InitMatchmaker()

  var adminUser models.User
	result := all.Postgres.First(&adminUser, "access_level = ?", 2)
	
	if result.RowsAffected != 0 {
		return
	}

	common.CreateUser("admin", "admin", 2)
}

func main() {
  args := strings.Join(os.Args, ";")
  if strings.Contains(args, "-reset_admin_password") {
    var adminUser models.User
    result := all.Postgres.First(&adminUser, "access_level = ?", 2)

    if result.RowsAffected == 0 {
      fmt.Println("No admin user found!")
    }

    adminUser.Password = all.HashString("admin")
    all.Postgres.Save(&adminUser)

    fmt.Println("Admin password reset!")

    return
  }

  if strings.Contains(args, "-reset_database") {
    all.Postgres.Exec("DROP SCHEMA public CASCADE;")
    all.Postgres.Exec("CREATE SCHEMA public;")
    all.Postgres.Exec("GRANT ALL ON SCHEMA public TO postgres;")
    all.Postgres.Exec("GRANT ALL ON SCHEMA public TO public;")
    all.Postgres.Exec("COMMENT ON SCHEMA public IS 'standard public schema';")
    all.AutoMigrate()
    return
  }
  
  fmt.Println(args)

  r := gin.Default()

  r.Use(middleware.CheckDatabase)
  r.Use(middleware.AllowFromAnywhere)
  // r.Use(middleware.RateLimitMiddleware(30, 1))

  site := r.Group("/api")
  {
    site.POST("/user/login", controllers.UserLogin)
    site.POST("/user/create", middleware.RateLimitMiddleware(1, 1), controllers.UserCreate)
    site.POST("/user/refresh", controllers.SiteRefresh)
    site.POST("/user/update", middleware.VerifySiteToken, controllers.UserUpdate)
    site.GET("/user/locker", middleware.VerifySiteToken, controllers.UserGetLocker)

    site.GET("/admin/users", middleware.VerifySiteToken, controllers.AdminGetAllUsers)
    site.GET("/admin/locker/:accountId", middleware.VerifySiteToken, controllers.AdminGetLocker)
    site.POST("/admin/user/:accountId/give/admin", middleware.VerifySiteToken, controllers.AdminGiveUserAdmin)
    site.POST("/admin/user/:accountId/take/admin", middleware.VerifySiteToken, controllers.AdminTakeUserAdmin)
    site.GET("/admin/profile/accountId/:accountId/:profileId", middleware.VerifySiteToken, controllers.AdminGetProfile)
    site.POST("/admin/profile/accountId/:accountId", middleware.VerifySiteToken, controllers.AdminSaveProfile)
    site.POST("/admin/profile/accountId/:accountId/give/all", middleware.VerifySiteToken, controllers.AdminGiveAllSkins)
    site.POST("/admin/profile/accountId/:accountId/give/:itemId", middleware.VerifySiteToken, controllers.AdminGiveItem)
    site.POST("/admin/profile/accountId/:accountId/take/all", middleware.VerifySiteToken, controllers.AdminTakeAllSkins)
    site.POST("/admin/profile/accountId/:accountId/take/:itemId", middleware.VerifySiteToken, controllers.AdminTakeItem)
  }

  account := r.Group("/account/api")
  {
    account.POST("/oauth/token", controllers.OAuthMain)
    account.GET("/public/account", controllers.UserAccountPublic)
    account.GET("/public/account/displayName/:displayName", controllers.UserAccountPublicFromDisplayName)
    account.GET("/public/account/:accountId", middleware.VerifyAccessToken, controllers.UserAccountPrivate)
    account.GET("/public/account/:accountId/externalAuths", controllers.EmptyArray)
    account.DELETE("/oauth/sessions/kill/:token", middleware.VerifyAccessToken, controllers.KillSessionWithToken)
    account.DELETE("/oauth/sessions/kill", controllers.KillSession)
  }

  friends := r.Group("/friends/api")
  {
    friends.GET("/public/v1/:accountId/settings", middleware.VerifyAccessToken, controllers.EmptyObject)
    friends.GET("/public/friends/list/:accountId/recentPlayers", controllers.EmptyArray)
    friends.GET("/public/friends/:accountId", middleware.VerifyAccessToken, controllers.FriendsPublic)
    friends.POST("/public/friends/:accountId/:friendId", middleware.VerifyAccessToken, controllers.CreateFriend)
    friends.DELETE("/public/friends/:accountId/:friendId", middleware.VerifyAccessToken, controllers.DeleteFriend)
    friends.GET("/public/blocklist/:accountId", middleware.VerifyAccessToken, controllers.FriendsBlocked)
    friends.POST("/public/blocklist/:accountId/:friendId", middleware.VerifyAccessToken, controllers.BlockFriend)
    friends.DELETE("/public/blocklist/:accountId/:friendId", middleware.VerifyAccessToken, controllers.UnBlockFriend)
  }

  fortnite := r.Group("/fortnite/api")
  {
    fortnite.GET("/game/v2/profileToken/verify/*accountId", controllers.NoContent)

    fortnite.POST("/game/v2/profile/:accountId/client/:action", middleware.VerifyAccessToken, controllers.ProfileActionHandler)
    fortnite.POST("/game/v2/tryPlayOnPlatform/account/*accountId", middleware.VerifyAccessToken, controllers.True)
    fortnite.GET("/game/v2/enabled_features", middleware.VerifyAccessToken, controllers.EmptyArray)
    fortnite.GET("/receipts/v1/account/:accountId/receipts", middleware.VerifyAccessToken, controllers.EmptyArray)
    fortnite.GET("/storefront/v2/keychain", middleware.VerifyAccessToken, controllers.StorefrontKeychain)
    fortnite.GET("/calendar/v1/timeline", controllers.CalendarTimeline)
    fortnite.GET("/storefront/v2/catalog", controllers.StorefrontCatalog)

    fortnite.GET("/cloudstorage/system", controllers.SystemCloudFilesList)
    fortnite.GET("/cloudstorage/system/:fileName", controllers.SystemCloudFile)
    fortnite.GET("/cloudstorage/user/:accountId", middleware.VerifyAccessToken, controllers.UserCloudFilesList)
    fortnite.GET("/cloudstorage/user/:accountId/:fileName", middleware.VerifyAccessToken, controllers.UserCloudFile)
    fortnite.PUT("/cloudstorage/user/:accountId/ClientSettings.Sav", middleware.VerifyAccessToken, controllers.SaveUserCloudFile)

    fortnite.GET("/game/v2/matchmakingservice/ticket/player/:accountId", middleware.VerifyAccessToken, controllers.MatchmakingTicket)
    fortnite.GET("/game/v2/matchmaking/account/:accountId/session/:sessionId", middleware.VerifyAccessToken, controllers.GetMatchmakingKey)
    fortnite.GET("/matchmaking/session/:sessionId", middleware.VerifyAccessToken, controllers.GetMatchmakeSession)

    fortnite.POST("/matchmaking/zomb/server", middleware.ServerSecret, controllers.AddNewGameServer)
    fortnite.DELETE("/matchmaking/zomb/server", middleware.ServerSecret, controllers.RemoveGameServer)

    fortnite.GET("/fortnite/api/v2/versioncheck/Windows", controllers.UpdateCheck)
  }

  party := r.Group("/party/api/v1/Fortnite")
  {
    party.Use(middleware.VerifyAccessToken)

    party.POST("/parties", controllers.PartyPost)
    party.GET("/parties/:partyId", controllers.PartyGet)
    party.PATCH("/parties/:partyId", controllers.PartyPatch)
    party.PATCH("/parties/:partyId/members/:memberId/meta", controllers.PartyPatchMemberMeta)
    party.DELETE("/parties/:partyId/members/:memberId", controllers.PartyDeleteMember)


    party.GET("/user/:accountId", controllers.PartyGetUser)
    party.GET("/user/:accountId/pings/:friendId/parties", controllers.PartyGetFriendPartyPings)
  }

  blank := r.Group("/")
  {
    blank.GET("/content/api/pages/*contentPageName", controllers.ContentPage)
    blank.GET("/waitingroom/api/waitingroom", controllers.NoContent)
    blank.POST("/datarouter/*api", controllers.NoContent)
    blank.GET("/lightswitch/api/service/bulk/status", controllers.Lightswitch)
    blank.GET("/lightswitch/api/service/Fortnite/status", controllers.Lightswitch)
    blank.GET("/fortnite/api/game/v2/chat/:accountId/:chatRoomType/:area/pc", controllers.ChatRooms)
  }

  r.GET("/", controllers.XMPP)
  r.GET("/match", controllers.Matchmaker)
  r.GET("/api/count/players", controllers.XMPPClients)
  r.GET("/api/count/queue", controllers.MatchmakerClients)
  r.GET("/api/count/party", controllers.Parties)
  
  r.Static("/assets", "./public/assets")
  r.StaticFile("api.json", "./public/api.json")
  r.StaticFile("data.json", "./public/data.json")

  r.GET("/cid/:cid", func(c *gin.Context) {
    c.File("./public/custom_cid_preview/" + c.Param("cid") + ".png")
  })

  r.NoRoute(func(c *gin.Context) {
    c.File("./public/index.html")
  })

  r.Run()
}