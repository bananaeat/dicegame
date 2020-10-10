package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"time"
)

//Db ...
var Db *sql.DB

//InitDB ...
func InitDB() (err error) {
	dsn := "root:123456@tcp(192.168.20.163:3306)/dicegame"
	Db, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("open %s failed, err:%v\n", dsn, err)
		return err
	}

	err = Db.Ping()
	if err != nil {
		fmt.Printf("ping failed, err:%v\n", err)
		return err
	}

	Db.Ping()
	fmt.Println("Connection success")
	return nil
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

// DiceUser ...
type DiceUser struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Score    string `json:"score"`
}

//Rank ...
type Rank struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
}

// AddDiceUser ...
func AddDiceUser(user *DiceUser) error {
	rand.Seed(time.Now().UnixNano())
	var salt [32]byte
	rand.Read(salt[:])
	x16salt := fmt.Sprintf("%x", salt)
	h := sha256.New()
	h.Write([]byte(user.Password))
	h.Write(salt[:])
	password := fmt.Sprintf("%x", h.Sum(nil))
	sqlStr := "insert into users(email, username, password, salt) values (?, ?, ?, ?)"
	ret, err := Db.Exec(sqlStr, user.Email, user.Username, password, x16salt)
	if err != nil {
		fmt.Println("insert failed, error:", err)
		return err
	}
	id, _ := ret.LastInsertId()
	fmt.Println("insert success, User ID is", id)
	copy("./assets/imgs/profilepic/placeholder.jpg", "./assets/imgs/profilepic/"+strconv.Itoa(int(id))+".jpg")
	return nil
}

// VerifyDiceUser ...
func VerifyDiceUser(email, password string) (string, int, error) {
	var truepassword string
	var username string
	var userid int
	var saltstr string
	sqlStr := "select userid, username, password, salt from users where email = ?"
	err := Db.QueryRow(sqlStr, email).Scan(&userid, &username, &truepassword, &saltstr)
	if err != nil {
		fmt.Println("scan failed, err:", err)
		return "", 0, err
	}

	h := sha256.New()
	h.Write([]byte(password))
	salt, _ := hex.DecodeString(saltstr)
	h.Write(salt)
	password = fmt.Sprintf("%x", h.Sum(nil))

	if password != truepassword {
		return "", 0, errors.New("password not matched or email not existed")
	}
	return username, userid, nil
}

// ModifyPassword ...
func ModifyPassword(email, newpassword string) error {
	rand.Seed(time.Now().UnixNano())
	var salt [32]byte
	rand.Read(salt[:])
	x16salt := fmt.Sprintf("%x", salt)
	h := sha256.New()
	h.Write([]byte(newpassword))
	h.Write(salt[:])
	password := fmt.Sprintf("%x", h.Sum(nil))

	sqlStr := "update users set password=?, salt=? where email=?"
	_, err := Db.Exec(sqlStr, password, x16salt, email)
	if err != nil {
		fmt.Println("update failed, error:", err)
		return err
	}
	return nil
}

// VerifyEmail ...
func VerifyEmail(email string) error {
	sqlStr := "select email from users"
	rows, err := Db.Query(sqlStr)
	if err != nil {
		fmt.Println("query failed, err:", err)
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var qemail string
		err = rows.Scan(&qemail)
		if qemail == email {
			return errors.New("email already existed")
		}
	}

	reg, err := regexp.Compile(`\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`)
	if err != nil {
		fmt.Println("regexp complie failed, err:", err)
		return err
	}
	if !reg.MatchString(email) {
		return errors.New("email not in proper format")
	}
	return nil
}

//UpdateScore ...
func UpdateScore(email string, score int) (bool, error) {
	var oldscore int
	sqlStr := "select score from users where email = ?"
	err := Db.QueryRow(sqlStr, email).Scan(&oldscore)
	if err != nil {
		fmt.Println("query failed, err:", err)
		return false, err
	}

	if oldscore < score {
		sqlStr = "update users set score = ? where email = ?"
		ret, err := Db.Exec(sqlStr, score, email)
		if err != nil {
			fmt.Println("update failed, err:", err)
			return false, err
		}
		fmt.Println("score updated:", ret)
		return true, nil
	}
	return false, nil
}

//RankByScore ...
func RankByScore(username string, rank int) ([10]Rank, int, int, error) {
	var users [10]Rank
	sqlStr := "select username, score from users order by score desc limit ?, 10"
	rows, err := Db.Query(sqlStr, rank-1)
	if err != nil {
		fmt.Println("query1 failed, err:", err)
		return [10]Rank{}, 0, 0, nil
	}

	defer rows.Close()

	var i int = 0
	for rows.Next() {
		var user string
		var score int

		err = rows.Scan(&user, &score)
		if err != nil {
			fmt.Println("query2 failed, err:", err)
			return [10]Rank{}, 0, 0, nil
		}

		users[i] = Rank{Username: user, Score: score}
		i++
	}

	var score int
	var r int
	sqlStr = "select r, r.score from (select username, score, rank() over (order by score desc) r from users) r where r.username = ?"
	err = Db.QueryRow(sqlStr, username).Scan(&r, &score)
	if err != nil {
		fmt.Println("query3 failed, err:", err)
		return [10]Rank{}, 0, 0, nil
	}

	return users, r, score, nil
}
