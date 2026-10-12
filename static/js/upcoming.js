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
				
		oTable = new DataTable('#upcomingmatches', {
			responsive: true,
			pageLength: 20,
			language: {
				url: '//cdn.datatables.net/plug-ins/1.13.6/i18n/fr-FR.json',
			},	
			order: [[0, "asc"]],
			ajax: {
				'url': '/api/upcoming',
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
			$('#ics_export').remove()
			if(caption != 'Toutes'){
				var calendarLink = document.createElement('a');
				calendarLink.id = 'ics_export';
				calendarLink.href = '/ics/upcoming/' + encodeURIComponent(value);
				calendarLink.target = '_blank';
				calendarLink.textContent = 'Exporter calendrier';
				document.getElementById('team-filter').appendChild(calendarLink);
			}
			oTable.ajax.url('/api/upcoming/' + value).load();
            oTable.draw();
		});
		bindTeamLinks();
	});
	  
});
