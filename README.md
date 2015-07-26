Slothful Soda Tracker (Go)
==========================

###A slothy take on pizza delivery trackers
![Slothful Soda Tracker demo GIF](https://github.com/AndyHaskell/slothful-soda-tracker/blob/master/public/images/slothful-soda-tracker-demo.gif?raw=true)
###How to install:
* Clone the project into the `src` directory of your `GOPATH` and run `go install`
* In `views/index.html` add your Google Maps API key where it says `YOUR_GOOGLE_MAPS_API_KEY` in the `<script>` tag for getting the Google Maps API
* Then to start the server, run `go-slothful-soda-tracker` and go to `localhost:1123`

###How to use:
When you load the main page, the Google Map that appears will have a hibiscus flower on it, which
is your marker. You can move your position with the W, A, S, and D keys (W = north, A = west,
S = south, D = east), and your marker will move on both your screen and every other user's screen.
The red Google Maps markers show the positions of all other users.

###The web app's backstory:
Slothful Soda is a soda invented by sloths living at the Cambridge Fresh Pond. They invented the soda
to sell to Ultimate players at parks in the Boston Area, and to make hibiscus into the next great
flavor of sodas (sloths love hibiscus flowers). Unfortunately, sloths are really slow, so to deliver
their delicious hibiscus concoction to thirsty Ultimate players while the game is still going on, they
got some delivery quadcopters! And sure enough, it caught on with Ultimate players across Boston who
are now rehydrating slothfully!

But Ultimate players want to be able to see when one of the Slothful Soda delivery quadcopters is
getting close to their playing field so they know how many points to play to. So the sloths at the
Cambridge Fresh Pond started a socket.io server that lets you see the locations of all of the
quadcopters in the Slothful Soda delivery fleet.

While in this web app the W, A, S, and D keys are used for changing your position, this could just
as easily be done with other forms of navigation, such as GPS location (I chose WASD navigation to
make it easy to test from localhost and also because I'm a gamer).

##NOTE: Don't tell friends who are likely to believe satire news articles like the ones on The Onion about this repository.  Sloths don't live on the Fresh Pond and don't have quadcopters but everyone knows if sloths figure out how to use quadcopters it means the Apocalypse is here and we really don't need any more people theorizing about when the Apocalypse is.
