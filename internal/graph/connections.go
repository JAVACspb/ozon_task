package graph

import (
	"github.com/ozosanek/ozon_task/internal/domain"
	"github.com/ozosanek/ozon_task/internal/graph/model"
	"github.com/ozosanek/ozon_task/internal/pagination"
)

func listOptions(first int, after *model.Cursor) domain.ListOptions {
	var cursor *string
	if after != nil {
		value := string(*after)
		cursor = &value
	}

	return domain.ListOptions{
		First: first,
		After: cursor,
	}
}

func postConnection(page domain.Page[domain.Post]) *model.PostConnection {
	edges := make([]*model.PostEdge, 0, len(page.Items))
	for _, post := range page.Items {
		item := post
		edges = append(edges, &model.PostEdge{
			Node:   &item,
			Cursor: model.Cursor(pagination.EncodeCursor(item.CreatedAt, item.ID)),
		})
	}

	return &model.PostConnection{
		Edges:    edges,
		PageInfo: pageInfo(page.EndCursor, page.HasNextPage),
	}
}

func commentConnection(page domain.Page[domain.Comment]) *model.CommentConnection {
	edges := make([]*model.CommentEdge, 0, len(page.Items))
	for _, comment := range page.Items {
		item := comment
		edges = append(edges, &model.CommentEdge{
			Node:   &item,
			Cursor: model.Cursor(pagination.EncodeCursor(item.CreatedAt, item.ID)),
		})
	}

	return &model.CommentConnection{
		Edges:    edges,
		PageInfo: pageInfo(page.EndCursor, page.HasNextPage),
	}
}

func pageInfo(endCursor *string, hasNextPage bool) *model.PageInfo {
	var cursor *model.Cursor
	if endCursor != nil {
		value := model.Cursor(*endCursor)
		cursor = &value
	}

	return &model.PageInfo{
		HasNextPage: hasNextPage,
		EndCursor:   cursor,
	}
}
