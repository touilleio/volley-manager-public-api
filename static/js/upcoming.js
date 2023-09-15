var oTable;

$( document ).ready(function() {
    //console.log( "ready!" );
	oTable = new DataTable('#example', {
		pageLength: 15,
		language: {
			url: '//cdn.datatables.net/plug-ins/1.13.6/i18n/fr-FR.json',
		},	
		order: [[0, "desc"]],
		ajax: {
			'url': '/upcoming',
            'dataSrc': ''
        },
		columns: [
			{
				data: 'playDate',
				render: DataTable.render.datetime('DD.MM.YYYY HH:mm')
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
		select: false,
		initComplete: function (settings, json) {
			
			teams = []
			
			$.getJSON("/teams", function(result){
				$.each(result, function(i, field){
					teams.push(field + ":" + i)
				});
				
				teams.sort();
				teams.unshift("Toutes:0")
				
				$('#team-filter').append('<label>&nbsp; Equipe:</label>');
				$('#team-filter').append('<select class="form-control input-sm"  id="sel_team_id"></select>');
				
				for (var ele in teams) {
					var obj = teams[ele].split(":");
					$('#sel_team_id').append('<option value="' + obj[1] + '">' + obj[0] + '</option>');
				}
				
				// Filter results on select change
				$('#sel_team_id').on('change', function () {
					val = $(this).find(":selected").text()
					if(val == "Toutes")
						oTable.columns(2).search("").draw();
					else
						oTable.columns(2).search(val).draw();
				});
			});
		}
	});
});
