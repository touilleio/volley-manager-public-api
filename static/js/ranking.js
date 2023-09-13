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
			'url': '/ranking/6636',
            'dataSrc': ''
        },
		columns: [
			{
				data: 'teamCaption'
			},{
				data: 'rawGames'
			},{
				data: 'wins'
			},{
				data: 'defeats'
			},{
				data: 'points'
			}
		],
		dom: 'Bfrtip',
		select: false,
		initComplete: function (settings, json) {
			//get all different teams
			teams = []
			
			for (var ele in json) {
				//entry=json[ele].teams.home.teamId
				t = json[ele].teamCaption + ":" + json[ele].teamId
				if(!teams.includes(t) && t.includes("Gibloux")){
					teams.push(t)
				}
			}
			
			teams.sort();
			teams.unshift("Toutes:0")
			
			// select filter
			$('#team-filter').append('<label>&nbsp; Equipe:</label>');
			$('#team-filter').append('<select class="form-control input-sm"  id="sel_team_id"></select>');
			team_ids = [{0: 'Toutes'}, {11902: 'Gibloux Volley F2'}, {6636: 'Gibloux Volley H3'}];
			
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
		}
	});
});
