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
	ID        githubv4.ID
	CreatedAt time.Time
	Author    struct {
		Login string
	}
}

type discussionCommentWithReplies struct {
	discussionCommentNode
	Replies struct {
		PageInfo struct {
			HasPreviousPage bool
			StartCursor     githubv4.String
		}
		Nodes []discussionCommentNode
	} `graphql:"replies(last: 100)"`
}

type discussionCommentsQuery struct {
	Repository struct {
		Discussion struct {
			Comments struct {
				PageInfo struct {
					HasPreviousPage bool
					StartCursor     githubv4.String
				}
				Nodes []discussionCommentWithReplies
			} `graphql:"comments(last: 100, before: $commentsCursor)"`
		} `graphql:"discussion(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// discussionCommentRepliesQuery is used to paginate replies for a single comment.
type discussionCommentRepliesQuery struct {
	Node struct {
		DiscussionComment struct {
			Replies struct {
				PageInfo struct {
					HasPreviousPage bool
					StartCursor     githubv4.String
				}
				Nodes []discussionCommentNode
			} `graphql:"replies(last: 100, before: $cursor)"`
		} `graphql:"... on DiscussionComment"`
	} `graphql:"node(id: $id)"`
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
		since := r.SinceTime()
		var commentsCursor *githubv4.String
		for {
			variables["commentsCursor"] = commentsCursor
			var q discussionCommentsQuery
			if err := c.v4Client.Query(ctx, &q, variables); err != nil {
				return false, "", skipNotFound(err)
			}
			for _, comment := range q.Repository.Discussion.Comments.Nodes {
				if matched, _ := c.matchComment(comment.discussionCommentNode, since, skipUserFilter, r); matched {
					return true, "", nil
				}
				for _, reply := range comment.Replies.Nodes {
					if matched, _ := c.matchComment(reply, since, skipUserFilter, r); matched {
						return true, "", nil
					}
				}
				if comment.Replies.PageInfo.HasPreviousPage {
					matched, err := c.paginateReplies(ctx, comment.ID, comment.Replies.PageInfo.StartCursor, since, skipUserFilter, r)
					if err != nil {
						return false, "", err
					}
					if matched {
						return true, "", nil
					}
				}
			}
			if !q.Repository.Discussion.Comments.PageInfo.HasPreviousPage {
				break
			}
			cursor := q.Repository.Discussion.Comments.PageInfo.StartCursor
			commentsCursor = &cursor
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

func (c *DiscussionChecker) paginateReplies(ctx context.Context, commentID githubv4.ID, startCursor githubv4.String, since time.Time, skipUserFilter bool, r *rule.WatchRule) (bool, error) {
	cursor := &startCursor
	for {
		variables := map[string]any{
			"id":     commentID,
			"cursor": cursor,
		}
		var q discussionCommentRepliesQuery
		if err := c.v4Client.Query(ctx, &q, variables); err != nil {
			return false, skipNotFound(err)
		}
		for _, reply := range q.Node.DiscussionComment.Replies.Nodes {
			if matched, _ := c.matchComment(reply, since, skipUserFilter, r); matched {
				return true, nil
			}
		}
		if !q.Node.DiscussionComment.Replies.PageInfo.HasPreviousPage {
			break
		}
		next := q.Node.DiscussionComment.Replies.PageInfo.StartCursor
		cursor = &next
	}
	return false, nil
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
