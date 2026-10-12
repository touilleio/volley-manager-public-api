var oTable;

function capitalizeFirstLetter(string) {
    return string.charAt(0).toUpperCase() + string.slice(1);
}

function escapeHtml(value) {
	return $('<div>').text(value == null ? '' : value).html();
}

function renderLeagueCell(data, type, row) {
	if (type !== 'display') {
		return data;
	}
	if (!row || row.isCup !== true) {
		return escapeHtml(data);
	}

	return '<span class="league-cell"><span class="league-cell-text">' + escapeHtml(data) + '</span><span class="league-cup-indicator"><span data-feather="award" class="league-cup-icon" aria-hidden="true"></span><span class="league-cup-label">Coupe</span></span></span>';
}

function replaceFeatherIcons() {
	if (window.feather) {
		feather.replace();
	}
}

$( document ).ready(function() {
	
	$('.page_menu a').each(function(e) {
		if(window.location.pathname.includes($(this).attr('href'))){
			$(this).attr('class', 'active');
		}
    });
	
	$.getJSON("/api/teams", function(result){
		// select filter
		$('#team-filter').append('<label>&nbsp; Equipe:</label>');
		$('#team-filter').append('<select class="form-control input-sm"  id="sel_team_id"></select>');
		
		populateTeamOptions(result, 'Toutes');
		
		//console.log( "ready!" );
		oTable = new DataTable('#pastmatches', {
			responsive: true,
			pageLength: 20,
			language: {
				url: '//cdn.datatables.net/plug-ins/1.13.6/i18n/fr-FR.json',
			},	
			order: [[0, "desc"]],
			ajax: {
				'url': '/api/past',
				'dataSrc': ''
			},
			columnDefs: [
				{ type: "de_datetime", targets: 0 }
			],
			columns: [
				{
					data: 'playDate',
					render: function(data, type, full) {
						return capitalizeFirstLetter(moment(data, "YYYY-MM-DD HH:mm:ss").format("dddd DD.MM.YYYY HH:mm"))
					}
				},{
					data: 'homeTeam',
					render: DataTable.render.text()
				},{
					data: 'awayTeam',
					render: DataTable.render.text()
				},{
					data: 'phase',
					render: renderLeagueCell
				},{
					data: 'hall',
					render: DataTable.render.text()
				},{
					data: 'wonSetsHomeTeam',
					render: function(data, param, row) {
						return row.wonSetsHomeTeam + " / " + row.wonSetsAwayTeam
					}
				},{
					data: 'winner',
					render: function(data, param, row) {
						if(data == 'team_away'){
							return param === 'display' ? escapeHtml(row.awayTeam) : row.awayTeam
						}else if(data == 'team_home'){
							return param === 'display' ? escapeHtml(row.homeTeam) : row.homeTeam
						}else{
							return "à Communiquer"
						}
					}
				}
			],
			dom: 'Bfrtip',
			select: false,
			drawCallback: replaceFeatherIcons
		});
		oTable.on('responsive-display.dt', replaceFeatherIcons);
		
		// Filter results on select change
		$('#sel_team_id').on('change', function () {
			caption = $(this).find(":selected").text()
			value = $(this).find(":selected").val()
			oTable.ajax.url('/api/past/' + value).load();
            oTable.draw();
		});
		bindTeamLinks();
	});
	  
});
