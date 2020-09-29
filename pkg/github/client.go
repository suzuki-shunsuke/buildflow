package github

import (
	"context"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

type Client struct {
	Client *github.Client
}

type ParamsNew struct {
	Token string
}

func New(ctx context.Context, params ParamsNew) Client {
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: params.Token},
	))
	return Client{
		Client: github.NewClient(tc),
	}
}

type ParamsGetPR struct {
	Owner string
	Repo  string
	PRNum int
}

type ParamsGetPRFiles struct {
	Owner string
	Repo  string
	PRNum int
}

type ParamsListPRsWithCommit struct {
	Owner string
	Repo  string
	SHA   string
}

func (client Client) GetPR(ctx context.Context, params ParamsGetPR) (*github.PullRequest, *github.Response, error) {
	return client.Client.PullRequests.Get(ctx, params.Owner, params.Repo, params.PRNum)
}

func (client Client) GetPRFiles(ctx context.Context, params ParamsGetPRFiles) ([]*github.CommitFile, *github.Response, error) {
	return client.Client.PullRequests.ListFiles(ctx, params.Owner, params.Repo, params.PRNum, nil)
}

func (client Client) ListPRsWithCommit(ctx context.Context, params ParamsListPRsWithCommit) ([]*github.PullRequest, *github.Response, error) {
	return client.Client.PullRequests.ListPullRequestsWithCommit(ctx, params.Owner, params.Repo, params.SHA, nil)
}
