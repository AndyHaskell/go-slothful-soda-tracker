var appData = {
  yourID:  "",
  yourPos: null,
  ws:  null,
  mapData: {
    map:     null,
    markers: {}
  }
}

//Create a new Google Map centered at the Cambridge Fresh Pond
var initMap = function(){
  var opt = {
    center   : new google.maps.LatLng(FRESH_POND.lat, FRESH_POND.lng),
    zoom     : 13,
    mapTypeId: google.maps.MapTypeId.HYBRID
  }

  appData.mapData.map = new google.maps.Map(document.getElementById('map'), opt)
}

var FRESH_POND = {lat: 42.388282, lng: -71.153968}

//Generate a random location near the Cambridge Fresh Pond
var initLocation = function(){
  return new google.maps.LatLng(FRESH_POND.lat + (Math.random()*.01-.005),
                                FRESH_POND.lng + (Math.random()*.01-.005))
}

appData.ws = new WebSocket('ws://localhost:1123/ws')

appData.ws.onopen = function(ev){
  initMap()
  appData.yourPos = initLocation()
}

appData.ws.onclose = function(ev){
  console.log('Connection closed')
}

appData.ws.onmessage = function(ev){
  var msg     = JSON.parse(ev.data),
      msgType = msg.msgType,
      data    = msg.data

  var map = appData.mapData.map
  var markers = appData.mapData.markers

  if (msgType)
    if (msgType == "Your ID") {
      appData.yourID = appData.yourID == "" ? data : appData.yourID
      var pos = appData.yourPos
	
      appData.ws.send(JSON.stringify({lat: pos.lat(), lng: pos.lng()}))
    }
    else if (msgType == "Everyone") {
      var you = appData.yourID
      var users = data.users
      var ids = Object.keys(data.users)

      for (var i = 0; i < ids.length; i++) {
        if (users[ids[i]].id != you) {
          markers[users[ids[i]].id] = new google.maps.Marker({
            position :
			  new google.maps.LatLng(users[ids[i]].lat, users[ids[i]].lng),
            map      : appData.mapData.map
          })
        }
      }

      markers[you] = new google.maps.Marker({
        position: appData.yourPos,
        map     : map,
        icon    : {
          url       : 'images/hibiscus.png',
          scaledSize: new google.maps.Size(30, 30)
        }
      })
    } else if (msgType == "User joined") {
      markers[data] = null;
    } else if (msgType == "User disconnected") {
      if (markers[data] != null) {
        var id = data

        markers[id].setMap(null)
        delete(markers[id])
      }
    } else if (msgType == "Update coordinates") {
      var id = data.id

      var newPos = new google.maps.LatLng(data.lat, data.lng)
      if (markers[id] == null) {
        markers[id] = new google.maps.Marker({
          position: newPos,
          map     : map
        })
      } else
        markers[id].setPosition(newPos)
    }
}

$(document).keydown(WASD)