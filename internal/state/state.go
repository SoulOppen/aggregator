package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/SoulOppen/aggregator/internal/config"
	"github.com/SoulOppen/aggregator/internal/database"
	"github.com/SoulOppen/aggregator/internal/rrss"
	"github.com/google/uuid"
)

type State struct {
	Db     *database.Queries
	Config *config.Config
}
type Command struct {
	Name string
	Args []string
}
type Commands struct {
	Callbacks map[string]func(*State, Command) error
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("the login handler expects a single argument, the username")
	} else if len(cmd.Args) == 1 {
		_, err := s.Db.GetUser(context.Background(), sql.NullString{String: cmd.Args[0], Valid: true})
		if err != nil {
			return err
		}
		s.Config.SetUser(cmd.Args[0])
		fmt.Printf("User set to %s\n", cmd.Args[0])
	} else {
		return errors.New("the login handler expects a single argument, the username")
	}
	return nil
}
func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Args) == 0 || len(cmd.Args) > 2 {
		return errors.New("the register handler expects a single argument, the username")
	}

	_, err := s.Db.GetUser(context.Background(), sql.NullString{String: cmd.Args[0], Valid: true})
	if err == nil {
		return errors.New("registro ya existe")
	}
	var newUser database.CreateUserParams
	newUser.ID = uuid.New()
	newUser.Name = sql.NullString{String: cmd.Args[0], Valid: true}
	newUser.CreatedAt = time.Now()
	newUser.UpdatedAt = time.Now()
	createdUser, err := s.Db.CreateUser(context.Background(), newUser)
	if err != nil {
		return errors.New("Failed")
	}
	fmt.Printf("user created %s\n", createdUser.Name.String)
	s.Config.SetUser(createdUser.Name.String)
	return nil

}
func HandlerReset(s *State, cmd Command) error {
	err := s.Db.DeleteAllUsers(context.Background())
	if err != nil {
		return errors.New("no se pudo borrar")
	}
	return nil

}
func HandlerGetUser(s *State, cmd Command) error {
	current, ok := s.Config.Config["current_user_name"]
	if !ok {
		return errors.New("no hay usuario")
	}
	names, err := s.Db.GetAllUsers(context.Background())
	if err != nil {
		return errors.New("no se pudo acceder a usuarios")
	}
	for _, name := range names {
		if name.String == current {
			fmt.Printf("* %s (current)\n", name.String)
		} else {
			fmt.Printf("* %s\n", name.String)
		}
	}
	return nil

}
func HandlerAgg(s *State, cmd Command) error {
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return fmt.Errorf("usage: %v <time_between_reqs>", cmd.Name)
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	log.Printf("Collecting feeds every %s...", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)

	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func scrapeFeeds(s *State) {
	feed, err := s.Db.GetNextFeedToFetch(context.Background())
	if err != nil {
		log.Println("Couldn't get next feeds to fetch", err)
		return
	}
	log.Println("Found a feed to fetch!")
	scrapeFeed(s.Db, feed)
}

func scrapeFeed(db *database.Queries, feed database.Feed) {
	_, err := db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		log.Printf("Couldn't mark feed %s fetched: %v", feed.Name.String, err)
		return
	}

	feedData, err := rrss.FetchFeed(context.Background(), feed.Url.String)
	if err != nil {
		log.Printf("Couldn't collect feed %s: %v", feed.Name.String, err)
		return
	}
	for _, item := range feedData.Channel.Item {
		publishedAt := sql.NullTime{}
		if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
			publishedAt = sql.NullTime{
				Time:  t,
				Valid: true,
			}
		}

		_, err = db.CreatePost(context.Background(), database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			FeedID:    feed.ID,
			Title:     item.Title,
			Description: sql.NullString{
				String: item.Description,
				Valid:  true,
			},
			Url:         item.Link,
			PublishedAt: publishedAt,
		})
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				continue
			}
			log.Printf("Couldn't create post: %v", err)
			continue
		}
	}
	log.Printf("Feed %s collected, %v posts found", feed.Name.String, len(feedData.Channel.Item))

}
func AddFeed(s *State, args Command, user database.User) error {
	if len(args.Args) < 2 {
		return errors.New("faltan argumentos: nombre y URL")
	}

	data, err := s.Db.GetUser(context.Background(), sql.NullString{String: user.Name.String, Valid: true})
	if err != nil {
		return err
	}

	newFeed := database.FeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name: sql.NullString{
			String: args.Args[0],
			Valid:  true,
		},
		Url: sql.NullString{
			String: args.Args[1],
			Valid:  true,
		},
		UserID: uuid.NullUUID{
			UUID:  data.ID,
			Valid: true,
		},
	}
	feed, err := s.Db.Feed(context.Background(), newFeed)
	if err != nil {
		return err
	}
	feedFollow, err := s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    data.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("couldn't create feed follow: %w", err)
	}
	fmt.Println("Feed created successfully:")
	printFeed(feed, data)
	fmt.Println()
	fmt.Println("Feed followed successfully:")
	printFeedFollow(feedFollow.UserName.String, feedFollow.FeedName.String)
	fmt.Println("=====================================")
	return nil
}
func HandlerFeed(s *State, args Command) error {
	data, err := s.Db.GetAllFeeds(context.Background())
	if err != nil {
		return errors.New("no se registro")
	}

	for _, item := range data {
		fmt.Printf("%s\n", item.Name.String)
		fmt.Printf("%s\n", item.Url.String)
		name, err := s.Db.GetUserName(context.Background(), item.UserID.UUID)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", name.String)
	}
	return nil
}
func HandlerFollow(s *State, cmd Command, user database.User) error {

	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <feed_url>", cmd.Name)
	}

	feed, err := s.Db.GetFeedByURL(context.Background(), sql.NullString{String: cmd.Args[0], Valid: true})
	if err != nil {
		return fmt.Errorf("couldn't get feed: %w", err)
	}

	ffRow, err := s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("couldn't create feed follow: %w", err)
	}

	fmt.Println("Feed follow created:")
	printFeedFollow(ffRow.UserName.String, ffRow.FeedName.String)
	return nil
}

func HandlerListFeedFollows(s *State, cmd Command, user database.User) error {

	feedFollows, err := s.Db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("couldn't get feed follows: %w", err)
	}

	if len(feedFollows) == 0 {
		fmt.Println("No feed follows found for this user.")
		return nil
	}

	fmt.Printf("Feed follows for user %s:\n", user.Name.String)
	for _, ff := range feedFollows {
		fmt.Printf("* %s\n", ff.FeedName.String)
	}

	return nil
}
func HandlerUnfollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("usage: %s <feed_url>", cmd.Name)
	}

	feed, err := s.Db.GetFeedByURL(
		context.Background(),
		sql.NullString{
			String: cmd.Args[0],
			Valid:  true,
		},
	)
	if err != nil {
		return fmt.Errorf("couldn't get feed: %w", err)
	}

	err = s.Db.DelFeedFollow(
		context.Background(),
		database.DelFeedFollowParams{UserID: user.ID, FeedID: feed.ID},
	)
	if err != nil {
		return fmt.Errorf("couldn't unfollow feed: %w", err)
	}

	fmt.Printf("Successfully unfollowed feed: %s\n", feed.Name.String)
	return nil
}
func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		current, ok := s.Config.Config["current_user_name"]
		if !ok || current == "" {
			return errors.New("no hay usuario logueado")
		}
		user, err := s.Db.GetUser(
			context.Background(),
			sql.NullString{String: current, Valid: true},
		)
		if err != nil {
			return fmt.Errorf("usuario inválido: %w", err)
		}
		return handler(s, cmd, user)
	}
}
func HandlerBrowse(s *State, cmd Command, user database.User) error {
	limit := 2
	if len(cmd.Args) == 1 {
		if specifiedLimit, err := strconv.Atoi(cmd.Args[0]); err == nil {
			limit = specifiedLimit
		} else {
			return fmt.Errorf("invalid limit: %w", err)
		}
	}

	posts, err := s.Db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("couldn't get posts for user: %w", err)
	}

	fmt.Printf("Found %d posts for user %s:\n", len(posts), user.Name.String)
	for _, post := range posts {
		fmt.Printf("%s from %s\n", post.PublishedAt.Time.Format("Mon Jan 2"), post.FeedName.String)
		fmt.Printf("--- %s ---\n", post.Title)
		fmt.Printf("    %v\n", post.Description.String)
		fmt.Printf("Link: %s\n", post.Url)
		fmt.Println("=====================================")
	}

	return nil
}

func printFeed(feed database.Feed, user database.User) {
	fmt.Printf("* ID:            %s\n", feed.ID)
	fmt.Printf("* Created:       %v\n", feed.CreatedAt)
	fmt.Printf("* Updated:       %v\n", feed.UpdatedAt)
	fmt.Printf("* Name:          %s\n", feed.Name.String)
	fmt.Printf("* URL:           %s\n", feed.Url.String)
	fmt.Printf("* User:          %s\n", user.Name.String)
	fmt.Printf("* LastFetchedAt: %v\n", feed.LastFetchedAt.Time)
}
func printFeedFollow(username, feedname string) {
	fmt.Printf("* User:          %s\n", username)
	fmt.Printf("* Feed:          %s\n", feedname)
}

func (c *Commands) Run(s *State, cmd Command) error {
	cb, ok := c.Callbacks[cmd.Name]
	if !ok {
		return errors.New("not function")
	}
	err := cb(s, cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *Commands) Register(name string, f func(*State, Command) error) {
	if c.Callbacks == nil {
		c.Callbacks = make(map[string]func(*State, Command) error)
	}
	if _, exists := c.Callbacks[name]; !exists {
		c.Callbacks[name] = f
	}
}
