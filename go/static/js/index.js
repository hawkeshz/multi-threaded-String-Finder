$.fn.toggleState = function(b) {
	$(this).stop().animate({
		width: b ? "300px" : "50px"
	}, 600, "easeOutElastic" );
}

$(document).ready(function() {
	var container = $(".container");
	var boxContainer = $(".search-box-container");
	var submit = $(".submit");
	var searchBox = $(".search-box");
	var response = $(".response");
	var nodesHolder = $("#nodesHolder");
	var isOpen = false;
	submit.on("mousedown", function(e) {
		e.preventDefault();
		boxContainer.toggleState(!isOpen);
		isOpen = !isOpen;
		if(!isOpen) {
			handleRequest();
		} else {
			searchBox.focus();
		}
	});
	searchBox.keypress(function(e) {
		if(e.which === 13) {
			boxContainer.toggleState(false);
			isOpen = false;
			handleRequest();
		}
	});
	searchBox.blur(function() {
		boxContainer.toggleState(false);
		isOpen = false;
	});

	function displayResult(searchString, data) {
		var found = data.result
		if (found) {
			response.text(('"' + searchString + '" is FOUND by Node ' + data.node));
		} else {
			response.text(('"' + searchString + '" is NOT FOUND'));
		}
		response.animate({
			opacity: 1
		}, 300);
	}

	function handleRequest() {
		// You could do an ajax request here...
		var value = searchBox.val();
		searchBox.val('');
		if(value.length > 0) {
			$.post( "/search", { query: value}, function (data) {
				displayResult(value, data);
			});

		}
	}

	function getNodes() {
		$.post("/nodes", function(data) {
			if (data.length == 0) {
				nodesHolder.html('<h2 class="nonode">No nodes connected to server right now.</h2>');
			} else {
				var nodeHtml = "";
				//sorting by names
				data = data.sort(function(a, b) {
					return a.name.toString().localeCompare(b.name.toString());
				});
				// displaying nodes
				data.forEach(function (anode) {
					nodeHtml += '<div class="nodebox"><div class="nodeicon">';
					nodeHtml += '<img src="/static/assets/noode.png"/></div><div class="nodeinfo">';
				  nodeHtml += '<h2>Node '+anode.name+'</h2>';
					nodeHtml += '<h5>Files: <span class="chunkfiles">'+anode.chunks+'</span></h5>';
					nodeHtml += '<h5>Files in memory: <span class="chunkfiles">'+anode.memory+'</span></h5>';
					nodeHtml += '</div></div></div>';
				});
				nodesHolder.html(nodeHtml);
			}
		})
	}

	getNodes()
	window.setInterval(function(){
	  getNodes()
	}, 5000);


});
