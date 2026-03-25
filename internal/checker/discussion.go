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

type discussionQuery struct {
	Repository struct {
		Discussion struct {
			Closed     bool
			IsAnswered bool
		} `graphql:"discussion(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

type discussionCommentNode struct {
	CreatedAt time.Time
	Author    struct {
		Login string
	}
}

type discussionCommentsQuery struct {
	Repository struct {
		Discussion struct {
			Comments struct {
				Nodes []struct {
					discussionCommentNode
					Replies struct {
						Nodes []discussionCommentNode
					} `graphql:"replies(last: 100)"`
				}
			} `graphql:"comments(last: 100)"`
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
			if matched, _ := c.matchComment(comment.discussionCommentNode, since, skipUserFilter, r); matched {
				return true, "", nil
			}
			for _, reply := range comment.Replies.Nodes {
				if matched, _ := c.matchComment(reply, since, skipUserFilter, r); matched {
					return true, "", nil
				}
			}
		}
		return false, "", nil
	case "closed", "answered":
		var q discussionQuery
		if err := c.v4Client.Query(ctx, &q, variables); err != nil {
			return false, "", skipNotFound(err)
		}
		d := q.Repository.Discussion
		matched := (cond == "closed" && d.Closed) || (cond == "answered" && d.IsAnswered)
		if !matched {
			return false, "", nil
		}
		return true, "true", nil
	}
	return false, "", nil
}

func (c *DiscussionChecker) matchComment(node discussionCommentNode, since time.Time, skipUserFilter bool, r *rule.WatchRule) (bool, string) {
	if !node.CreatedAt.After(since) {
		return false, ""
	}
	if !skipUserFilter && shouldIgnoreUser(c.currentUser, r.CompiledIgnoreUsers(), node.Author.Login) {
		return false, ""
	}
	return true, ""
}
