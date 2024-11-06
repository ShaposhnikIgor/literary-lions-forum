package models

// import (

// 	//"literary-lions/internal/database"

// )

// func LikePost(postID, userID int) error {
// 	stmt, err := DB.Prepare("INSERT INTO likes (post_id, user_id) VALUES (?, ?)")
// 	if err != nil {
// 		return err
// 	}
// 	_, err = stmt.Exec(postID, userID)
// 	return err
// }

// func DislikePost(postID, userID int) error {
// 	stmt, err := DB.Prepare("DELETE FROM likes WHERE post_id = ? AND user_id = ?")
// 	if err != nil {
// 		return err
// 	}
// 	_, err = stmt.Exec(postID, userID)
// 	return err
// }
