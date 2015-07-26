//Updates your current position, returning the position as a Google Maps LatLng.
var updatePosition = function(key){
  var pos = appData.yourPos
  var lat = pos.lat(), lng = pos.lng()

  if (key == "87") //W; move north
    lat += 0.00014
  if (key == "65") //A; move west
    lng -= 0.00020
  if (key == "83") //S; move south
    lat -= 0.00014
  if (key == "68") //D; move east
    lng += 0.00020

  return new google.maps.LatLng(lat, lng)
}

//WASD navigation; update your marker and your position on the server side.
var WASD = function(ev){
  var key = ev.which
  
  if (key == "87" || key == "65" || key == "83" || key == "68") {
    var yourPos = appData.yourPos = updatePosition(key)
    var yourId  = appData.yourID

    appData.mapData.markers[yourId].setPosition(yourPos)
    appData.mapData.map.panTo(yourPos)
  
    appData.ws.send(JSON.stringify(
      {id: yourId, lat: yourPos.lat(), lng: yourPos.lng()}))
  }
}