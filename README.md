# Strava Gear Hours

This small utility helps me determine when the suspension shocks on my bike
need servicing based on usage data from [Strava][].

[Strava]: https://www.strava.com/

Strava allows you to associate gear (e.g. a bike) with your rides and can
report the total distance they have been used for. However the service
intervals for most shocks are detailed in hours-used rather than distance
travelled.

## Install

Assuming that you have [Go][] installed, `GOPATH` setup, and this repo using
`go get`:

    go install

[Go]: https://golang.org

## Setup

1. Ensure that you have a [bike added][] and associated to activities in Strava.
1. Generate an API token by [creating an application][] in Strava.
1. Export the access token as an environment variable (noting preceding
   space to prevent it being stored in your shell history):

         export STRAVA_ACCESS_TOKEN=abc123

[bike added]: https://www.strava.com/settings/gear
[creating an application]: https://www.strava.com/settings/api

## Usage

If a component has been in use since you first started using the bike:

    strava-gear-hours -bike "my bike"

If you've serviced a component since you first started using the bike then
you can specify the date of last service:

    strava-gear-hours -bike "my bike" -since "2015-12-31"

## Notes

Strava allows you to associate components (shocks, chain, etc.) to gear.
That could provide a better way of keeping track of the last service. Except
that components aren't currently exposed by the [Strava API][].

[Strava API]: https://strava.github.io/api/
