// A module for discovering files that have been changed between two Git refs.
// Useful for CI in pull requests in monorepos.
package main

import (
	"context"
	"fmt"

	"dagger/git-affected/internal/dagger"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

type GitFilesChanged struct{}

func (m *GitFilesChanged) Files(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	// +default="HEAD"
	headRef string,
	// +optional
	// +default="main"
	baseRef string,
) ([]string, error) {
	repoDir := "/repo"

	_, err := source.Export(ctx, repoDir)
	if err != nil {
		return nil, err
	}

	files, err := diff(ctx, repoDir, headRef, baseRef)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func diff(ctx context.Context, dir string, headRef, baseRef string) ([]string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, fmt.Errorf("error opening repository: %w", err)
	}

	headTree, err := treeFromRef(repo, headRef)
	if err != nil {
		return nil, fmt.Errorf("error getting tree from head ref: %w", err)
	}

	baseTree, err := treeFromRef(repo, baseRef)
	if err != nil {
		return nil, fmt.Errorf("error getting tree from base ref: %w", err)
	}

	changes, err := object.DiffTree(headTree, baseTree)
	if err != nil {
		return nil, fmt.Errorf("error diffing trees: %w", err)
	}

	var files []string
	for _, ch := range changes {
		action, err := ch.Action()
		if err != nil {
			return nil, fmt.Errorf("error getting change action: %w", err)
		}

		var file string

		switch action {
		case merkletrie.Insert:
			file = ch.To.Name
		case merkletrie.Modify:
			file = ch.To.Name
		case merkletrie.Delete:
			file = ch.From.Name
		}

		files = append(files, file)
	}

	return files, nil
}

func treeFromRef(repo *git.Repository, ref string) (*object.Tree, error) {
	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, err
	}

	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return nil, err
	}

	return commit.Tree()
}
