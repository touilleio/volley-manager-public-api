var oTable;

function capitalizeFirstLetter(string) {
    return string.charAt(0).toUpperCase() + string.slice(1);
}

$( document ).ready(function() {
	
	$('.page_menu a').each(function(e) {
		if(window.location.pathname.includes($(this).attr('href'))){
			$(this).attr('class', 'active');
		}
    });
	
	teams = []
	
	$.getJSON("/teams", function(result){
		$.each(result, function(i, field){
			teams.push(field + ":" + i)
		});
		
		teams.sort();
		teams.unshift("Toutes:")
		
		// select filter
		$('#team-filter').append('<label>&nbsp; Equipe:</label>');
		$('#team-filter').append('<select class="form-control input-sm"  id="sel_team_id"></select>');
		
		for (var ele in teams) {
			var obj = teams[ele].split(":");
			$('#sel_team_id').append('<option value="' + obj[1] + '">' + obj[0] + '</option>');
		}
		
		//console.log( "ready!" );
		oTable = new DataTable('#pastmatches', {
			responsive: true,
			pageLength: 20,
			language: {
				url: '//cdn.datatables.net/plug-ins/1.13.6/i18n/fr-FR.json',
			},	
			order: [[0, "desc"]],
			ajax: {
				'url': '/past',
				'dataSrc': ''
			},
			columns: [
				{
					data: 'playDate',
					render: function(data, type, full) {
						return capitalizeFirstLetter(moment(data, "YYYY-MM-DD HH:mm:ss").format("dddd DD.MM.YYYY HH:mm"))
					}
				},{
					data: 'homeTeam'
				},{
					data: 'awayTeam'
				},{
					data: 'phase'
				},{
					data: 'hall'
				},{
					data: 'wonSetsHomeTeam',
					render: function(data, param, row) {
						return row.wonSetsHomeTeam + " / " + row.wonSetsAwayTeam
					}
				},{
					data: 'winner',
					render: function(data, param, row) {
						if(data == 'team_away'){
							return row.awayTeam
						}else if(data == 'team_home'){
							return row.homeTeam
						}else{
							return "à Communiquer"
						}
					}
				}
			],
			dom: 'Bfrtip',
			select: false
		});
		
		// Filter results on select change
		$('#sel_team_id').on('change', function () {
			caption = $(this).find(":selected").text()
			value = $(this).find(":selected").val()
			oTable.ajax.url('/past/' + value).load();
            oTable.draw();
		});
	});
	  
});
