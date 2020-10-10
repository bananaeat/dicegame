package main

import (
	"dicegame/equipments"
	"dicegame/monsters"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dchest/captcha"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

//Max ...
func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

//Min ...
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

//MessageObject ...
type MessageObject struct {
	Type   string      `json:"type"`
	Object interface{} `json:"object"`
}

//PlayerStatus ...
type PlayerStatus struct {
	HP           int           `json:"hp"`
	ATK          int           `json:"atk"`
	DEF          int           `json:"def"`
	WeaponID     int           `json:"weaponid"`
	ArmorID      int           `json:"armorID"`
	ReRollNum    int           `json:"rerollnum"`
	WeaponDices  map[int][]int `json:"weapondices"`
	ArmorDices   map[int][]int `json:"armordices"`
	Score        int           `json:"score"`
	RerollChance int           `json:"rerollchance"`
}

//MonsterStatus ...
type MonsterStatus struct {
	HP    int           `json:"hp"`
	ATK   int           `json:"atk"`
	DEF   int           `json:"def"`
	ID    int           `json:"id"`
	Name  string        `json:"name"`
	Dices map[int][]int `json:"dices"`
}

//CombatResult ...
type CombatResult struct {
	Player  PlayerStatus  `json:"player"`
	Monster MonsterStatus `json:"monster"`
}

//Equipment ...
type Equipment struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
}

//WinResult ...
type WinResult struct {
	NewMonsterID    int         `json:"newmonsterid"`
	NewMonsterHP    int         `json:"newmonsterhp"`
	NewEquipmentsID []Equipment `json:"newequipmentsid"`
}

//Reroll ...
type Reroll struct {
	Player         PlayerStatus `json:"player"`
	RerollDice     int          `json:"rerolldice"`
	RerollDiceSide int          `json:"rerolldiceside"`
}

var winResult WinResult

func generateWinResult(player PlayerStatus) WinResult {

	lenM := 0
	for _, m := range monsters.Monsters {
		if m.Score <= player.Score {
			lenM++
		}
	}
	lenM = Max(lenM, 1)

	rand.Seed(time.Now().UnixNano() + 1)
	mID := rand.Intn(lenM)
	mHP := monsters.Monsters[mID].HP

	var Es []Equipment
	lenW := 0
	for _, w := range equipments.Weapons {
		if w.Score <= player.Score {
			lenW++
		}
	}

	lenA := 0
	for _, a := range equipments.Armors {
		if a.Score <= player.Score {
			lenA++
		}
	}

	fmt.Println("lenA=", lenA, "lenW=", lenW)
	for i := 0; i < 2; i++ {
		var t string
		var eID int
		rand.Seed(time.Now().UnixNano() - int64(i))
		if rand.Intn(2) == 0 {
			t = "weapon"
			rand.Seed(time.Now().UnixNano() - int64(10*i))
			eID = rand.Intn(lenW)
		} else {
			t = "armor"
			rand.Seed(time.Now().UnixNano() - int64(10*i))
			eID = rand.Intn(lenA)
		}
		Es = append(Es, Equipment{t, eID})
	}
	return WinResult{mID, mHP, Es}
}

var userconn []*websocket.Conn

func authUser(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		d := struct {
			CaptchaID string
		}{
			captcha.New(),
		}
		c.HTML(http.StatusOK, "login.html", d)
		c.Abort()
		return
	}
	c.Next()
	return
}

func main() {
	if InitDB() != nil {
		fmt.Println("Database initialisation failed")
		return
	}

	//Load Resources
	r := gin.Default()
	r.LoadHTMLGlob("./assets/html/*.html")
	r.Static("/js", "./assets/js")
	r.Static("/css", "./assets/css")
	r.Static("/imgs", "./assets/imgs")
	r.Static("/audio", "./assets/audio")
	r.Static("/v1/assets", "./assets")
	r.StaticFile("/favicon.ico", "./assets/imgs/favicon.ico")

	//Init Data
	equipments.InitArmors()
	equipments.InitWeapons()
	monsters.InitMonsters()

	store := cookie.NewStore([]byte("bananaeat"))

	r.Use(sessions.Sessions("sessionid", store))

	r.GET("/login", func(c *gin.Context) {
		d := struct {
			CaptchaID string
		}{
			captcha.New(),
		}
		c.HTML(http.StatusOK, "login.html", d)
	})

	r.GET("/captcha/:img", func(c *gin.Context) {
		captcha.Server(captcha.StdWidth, captcha.StdHeight).ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/processcaptcha", func(c *gin.Context) {
		captchaID := c.Query("captchaID")
		captchasolution := c.Query("captchasolution")
		if !captcha.VerifyString(captchaID, captchasolution) {
			c.JSON(http.StatusOK, gin.H{
				"status": "failed",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
			})
		}
	})

	r.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", nil)
	})

	r.GET("/logout", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Delete("user")
		session.Delete("userid")
		session.Delete("email")
		session.Delete("createdat")
		session.Save()
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	r.POST("/register", func(c *gin.Context) {
		email := c.PostForm("email")
		username := c.PostForm("username")
		password := c.PostForm("password")
		fmt.Println(email, username, password)
		user := DiceUser{
			Email:    email,
			Username: username,
			Password: password,
		}
		err := AddDiceUser(&user)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "failed",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
			})
		}
	})

	r.POST("/login", func(c *gin.Context) {
		email := c.PostForm("email")
		password := c.PostForm("password")
		fmt.Println(email, password)
		username, userid, err := VerifyDiceUser(email, password)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "failed",
			})
		} else {
			session := sessions.Default(c)
			session.Set("email", email)
			session.Set("user", username)
			session.Set("userid", userid)
			session.Save()
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
			})
		}
	})

	v1 := r.Group("/v1")
	v1.Use(authUser)
	{
		v1.GET("/mainmenu", func(c *gin.Context) {
			c.HTML(http.StatusOK, "mainmenu.html", nil)
		})

		v1.GET("/dicegame", func(c *gin.Context) {
			c.HTML(http.StatusOK, "dicegame.html", nil)
		})

		v1.GET("/game", func(c *gin.Context) {
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				fmt.Println(err)
				return
			}
			player := PlayerStatus{HP: 5, ATK: 0, DEF: 0, WeaponID: 0, ArmorID: 0, ReRollNum: 2, Score: 0, RerollChance: 1}
			monster := MonsterStatus{HP: 3, ATK: 0, DEF: 0, ID: 0, Name: "史莱姆"}
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					fmt.Println(err)
					return
				}

				fmt.Println(string(msg))
				if string(msg) == "roll" {
					player.WeaponDices = make(map[int][]int)
					player.ArmorDices = make(map[int][]int)
					if player.ReRollNum > 0 {
						ATK := 0
						DEF := 0
						for i, w := range equipments.Weapons[player.WeaponID].Dices {
							rand.Seed(time.Now().UnixNano() + int64(i))
							d := rand.Intn(6)
							ATK += w[d]
							player.WeaponDices[player.WeaponID] = append(player.WeaponDices[player.WeaponID], d)
						}
						for i, a := range equipments.Armors[player.ArmorID].Dices {
							rand.Seed(time.Now().UnixNano() - int64(i))
							d := rand.Intn(6)
							DEF += a[d]
							player.ArmorDices[player.ArmorID] = append(player.ArmorDices[player.ArmorID], d)
						}
						player.ATK = ATK
						player.DEF = DEF
						player.ReRollNum--
						var msgobj = MessageObject{"playerstatus", player}
						conn.WriteJSON(msgobj)
						fmt.Println(msgobj)
					} else {
						var msgobj = MessageObject{"error", "无法再重掷骰子"}
						conn.WriteJSON(msgobj)
					}
				}

				if string(msg) == "confirm" {
					monster.Dices = make(map[int][]int)
					ATK := 0
					DEF := 0
					for _, w := range monsters.Monsters[monster.ID].Dices {
						rand.Seed(time.Now().UnixNano() + 10)
						d := rand.Intn(6)
						if w.Type == "ATK" {
							ATK += w.Dice[d]
						}
						if w.Type == "DEF" {
							DEF += w.Dice[d]
						}
						monster.Dices[monster.ID] = append(monster.Dices[monster.ID], d)
					}
					monster.ATK = ATK
					monster.DEF = DEF
					var msgobj = MessageObject{"monsterstatus", monster}
					conn.WriteJSON(msgobj)
					fmt.Println(msgobj)
					var combat = CombatResult{player, monster}
					combat.Player.HP -= Max((combat.Monster.ATK - combat.Player.DEF), 0)
					player.HP -= Max((combat.Monster.ATK - combat.Player.DEF), 0)
					combat.Monster.HP -= Max((combat.Player.ATK - combat.Monster.DEF), 0)
					monster.HP -= Max((combat.Player.ATK - combat.Monster.DEF), 0)
					msgobj = MessageObject{"combatresult", combat}
					conn.WriteJSON(msgobj)
					fmt.Println(msgobj)
					player.ReRollNum = 2
					if player.HP <= 0 {
						msgobj = MessageObject{"loseresult", player}
						conn.WriteJSON(msgobj)
						session := sessions.Default(c)
						email := session.Get("email").(string)
						ret, err := UpdateScore(email, player.Score)
						if err != nil {
							fmt.Println("update score failed")
						} else if ret == true {
							fmt.Println("updated score successfully")
						} else {
							fmt.Println("No need to update score")
						}
					}
					if monster.HP <= 0 {
						winResult = generateWinResult(player)
						player.Score += monsters.Monsters[monster.ID].Score
						msgobj = MessageObject{"winresult", winResult}
						conn.WriteJSON(msgobj)
						fmt.Println(msgobj)
						msgobj = MessageObject{"updatescore", player.Score}
						conn.WriteJSON(msgobj)
						fmt.Println(msgobj)
						monster.ID = winResult.NewMonsterID
						monster.Name = monsters.Monsters[monster.ID].Name
						monster.HP = monsters.Monsters[monster.ID].HP
						monster.ATK = 0
						monster.DEF = 0
						player.RerollChance = 1
						var msgobj = MessageObject{"monsterstatus", monster}
						conn.WriteJSON(msgobj)
						fmt.Println(msgobj)
					}
				}

				if strings.Split(string(msg), " ")[0] == "replace" {
					ind, _ := strconv.Atoi(strings.Split(string(msg), " ")[1])
					t := winResult.NewEquipmentsID[ind].Type
					i := winResult.NewEquipmentsID[ind].ID
					fmt.Println(t, i)
					if t == "weapon" {
						player.WeaponID = i
					}
					if t == "armor" {
						player.ArmorID = i
					}
					fmt.Println("replace", t, "to id=", i)
				}

				if strings.Split(string(msg), " ")[0] == "reroll" {
					if player.RerollChance > 0 {
						player.RerollChance--
						ind, _ := strconv.Atoi(strings.Split(string(msg), " ")[1])
						var d int
						var msgobj MessageObject
						if ind <= len(player.WeaponDices[player.WeaponID])-1 {
							player.ATK -= equipments.Weapons[player.WeaponID].Dices[ind][player.WeaponDices[player.WeaponID][ind]]
							rand.Seed(time.Now().UnixNano())
							d = rand.Intn(6)
							player.ATK += equipments.Weapons[player.WeaponID].Dices[ind][d]
							player.WeaponDices[player.WeaponID][ind] = d
							msgobj = MessageObject{"reroll", Reroll{player, ind, d}}
						} else {
							ind -= (len(player.WeaponDices[player.WeaponID]))
							player.DEF -= equipments.Armors[player.ArmorID].Dices[ind][player.ArmorDices[player.ArmorID][ind]]
							rand.Seed(time.Now().UnixNano())
							d = rand.Intn(6)
							player.DEF += equipments.Armors[player.ArmorID].Dices[ind][d]
							player.ArmorDices[player.ArmorID][ind] = d
							msgobj = MessageObject{"reroll", Reroll{player, ind + len(player.WeaponDices[player.WeaponID]), d}}
						}
						conn.WriteJSON(msgobj)
						fmt.Println(msgobj)
					} else {
						conn.WriteJSON(MessageObject{"error", "cannot reroll"})
					}
				}
			}
		})

		v1.GET("/loading", func(c *gin.Context) {
			c.HTML(http.StatusOK, "loading.html", nil)
		})

		v1.GET("/loadimgs", func(c *gin.Context) {
			var imgpathmonster []string
			filepath.Walk("./assets/imgs/monsters/", func(path string, info os.FileInfo, err error) error {
				imgpathmonster = append(imgpathmonster, path)
				return nil
			})

			var imgpathweapon []string
			filepath.Walk("./assets/imgs/weapons/", func(path string, info os.FileInfo, err error) error {
				imgpathweapon = append(imgpathweapon, path)
				return nil
			})

			var imgpatharmor []string
			filepath.Walk("./assets/imgs/armors/", func(path string, info os.FileInfo, err error) error {
				imgpatharmor = append(imgpatharmor, path)
				return nil
			})

			var imgs []string

			imgs = append(imgs, imgpathmonster[1:]...)
			imgs = append(imgs, imgpathweapon[1:]...)
			imgs = append(imgs, imgpatharmor[1:]...)

			c.JSON(http.StatusOK, imgs)
		})

		v1.GET("/loadaudios", func(c *gin.Context) {
			var audiopath []string
			filepath.Walk("./assets/audio", func(path string, info os.FileInfo, err error) error {
				audiopath = append(audiopath, path)
				return nil
			})
			audiopath = append(audiopath[:0], audiopath[1:]...)

			c.JSON(http.StatusOK, audiopath)
		})

		v1.GET("/ranklist", func(c *gin.Context) {
			session := sessions.Default(c)
			username := session.Get("user")
			c.HTML(http.StatusOK, "rank.html", username)
		})

		v1.GET("/rank", func(c *gin.Context) {
			username := c.Query("username")
			startrank, _ := strconv.Atoi(c.Query("startrank"))
			ranklist, rank, score, err := RankByScore(username, startrank)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{
					"status":   "ok",
					"ranklist": ranklist,
					"rank":     rank,
					"score":    score,
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"status": "failed",
				})
			}
		})
	}

	r.Run(":8000")
}
