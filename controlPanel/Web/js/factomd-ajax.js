function queryState(item, func) {
  var req = new XMLHttpRequest()

  req.onreadystatechange = function() {
    if(req.readyState == 4) {
      //console.log(item + " - " + req.response)
      func(req.response)
    }
  }
  req.open("GET", "/factomd?item=" + item, true)
  req.send()
}


$("#factom-search-submit").click(function() {
  var x = new XMLHttpRequest()
  x.onreadystatechange = function() {
    if(x.readyState == 4) {
      console.log(x.response)
      obj = JSON.parse(x.response)
      $("#changeme").text(obj.Type)
    }
  }
  var formData = new FormData();
  formData.append("method", "search")
  formData.append("search", $("#factom-search").val())

  x.open("POST", "/post")
  x.send(formData)
})

// Example Code to use for forms
/*
var form = document.getElementById("test_form")
form.addEventListener("submit", function(e) {
  e.preventDefault()
  var x = new XMLHttpRequest()

  x.onreadystatechange = function() {
    if(x.readyState == 4) {
      //console.log(x.response)
      //alert(x.response)
      $("#changeme").text(x.response)
    }
  }

  x.open("POST", "/post")
  x.send(new FormData(form))
})
*/