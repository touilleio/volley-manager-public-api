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
	//alert(moment.locale())
	//moment().locale('fr-fr')
	//alert(moment.locale())
	
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
		oTable = new DataTable('#upcomingmatches', {
			responsive: true,
			pageLength: 20,
			language: {
				url: '//cdn.datatables.net/plug-ins/1.13.6/i18n/fr-FR.json',
			},	
			order: [[0, "asc"]],
			ajax: {
				'url': '/upcoming',
				'dataSrc': ''
			},
			columnDefs: [
				{ type: "date", targets: 0 }
			],
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
				}
			],
			dom: 'Bfrtip',
			select: false
		});
		
		// Filter results on select change
		$('#sel_team_id').on('change', function () {
			caption = $(this).find(":selected").text()
			value = $(this).find(":selected").val()
			$('#ics_export').remove()
			if(caption != 'Toutes'){
				$('#team-filter').append('<a id="ics_export" href="/ics/upcoming/'+$('#sel_team_id').find(":selected").val()+'" target"_blank">Exporter calendrier</a>');
			}
			oTable.ajax.url('/upcoming/' + value).load();
            oTable.draw();
		});
	});
	  
});
