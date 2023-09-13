var oTable;

$( document ).ready(function() {
    //console.log( "ready!" );
	oTable = new DataTable('#example', {
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
				render: function(data) {
					return data
					var d = new Date(data);
					var datestring = d.getDate().toString().padStart(2, "0") + "." + (d.getMonth()+1).toString().padStart(2, "0") + "." + d.getFullYear() + " " + d.getHours().toString().padStart(2, "0") + ":" + d.getMinutes().toString().padStart(2, "0");
					return datestring;
				}
			},{
				data: 'teams',
				render: function(data) {
					return data.home.caption
				}
			},{
				data: 'teams',
				render: function(data) {
					return data.away.caption
				}
			},{
				data: 'phase',
				render: function(data) {
					return data.caption
				}
			},{
				data: 'hall',
				render: function(data) {
					return data.caption + ", " +" "+data.city
				}
			}
		],
		dom: 'Bfrtip',
		select: false,
		initComplete: function (settings, json) {
			//get all different teams
			teams = []
			
			for (var ele in json) {
				//entry=json[ele].teams.home.teamId
				t = json[ele].teams.home.caption + ":" + json[ele].teams.home.teamId
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
