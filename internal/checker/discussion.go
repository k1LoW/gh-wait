package checker

import (
	"context"
	"time"

	"github.com/k1LoW/gh-wait/internal/rule"
	"github.com/shurcooL/githubv4"
)

type DiscussionChecker struct {
	v4Client    *githubv4.Client
	currentUser string
}

func NewDiscussionChecker(v4Client *githubv4.Client, currentUser string) *DiscussionChecker {
	return &DiscussionChecker{v4Client: v4Client, currentUser: currentUser}
}

// discussionQuery fetches basic discussion fields (closed, answered, author).
type discussionQuery struct {
	Repository struct {
		Discussion struct {
			Closed     bool
			IsAnswered bool
			Author     struct {
				Login string
			}
		} `graphql:"discussion(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// discussionCommentsQuery fetches discussion comments for the "commented" condition.
type discussionCommentsQuery struct {
	Repository struct {
		Discussion struct {
			Comments struct {
				Nodes []struct {
					CreatedAt time.Time
					Author    struct {
						Login string
					}
				}
			} `graphql:"comments(first: 100)"`
		} `graphql:"discussion(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

func (c *DiscussionChecker) Check(ctx context.Context, r *rule.WatchRule) (bool, error) {
	return c.CheckConditions(ctx, r, r.Conditions)
}

func (c *DiscussionChecker) CheckConditions(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error) {
	return evalConditions(ctx, r, conditions, c.checkCondition, true)
}

func (c *DiscussionChecker) CheckState(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error) {
	return evalConditions(ctx, r, conditions, c.checkCondition, false)
}

func (c *DiscussionChecker) checkCondition(ctx context.Context, owner, repo string, r *rule.WatchRule, cond string, skipUserFilter bool) (bool, string, error) {
	variables := map[string]any{
		"owner":  githubv4.String(owner),
		"repo":   githubv4.String(repo),
		"number": githubv4.Int(int32(r.Number)), //nolint:gosec
	}

	switch cond {
	case "commented":
		var q discussionCommentsQuery
		if err := c.v4Client.Query(ctx, &q, variables); err != nil {
			return false, "", skipNotFound(err)
		}
		since := r.SinceTime()
		for _, comment := range q.Repository.Discussion.Comments.Nodes {
			if !comment.CreatedAt.After(since) {
				continue
			}
			if !skipUserFilter && shouldIgnoreUser(c.currentUser, r.CompiledIgnoreUsers(), comment.Author.Login) {
				continue
			}
			return true, "", nil
		}
		return false, "", nil
	case "closed":
		var q discussionQuery
		if err := c.v4Client.Query(ctx, &q, variables); err != nil {
			return false, "", skipNotFound(err)
		}
		if !q.Repository.Discussion.Closed {
			return false, "", nil
		}
		return true, "true", nil
	case "answered":
		var q discussionQuery
		if err := c.v4Client.Query(ctx, &q, variables); err != nil {
			return false, "", skipNotFound(err)
		}
		if !q.Repository.Discussion.IsAnswered {
			return false, "", nil
		}
		return true, "true", nil
	}
	return false, "", nil
}
