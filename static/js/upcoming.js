var oTable;

$( document ).ready(function() {
	
	teams = []
	
	$.getJSON("/teams", function(result){
		$.each(result, function(i, field){
			teams.push(field + ":" + i)
		});
		
		teams.sort();
		teams.unshift("Choisis une équipe...")
		
		// select filter
		$('#team-filter').append('<label>&nbsp; Equipe:</label>');
		$('#team-filter').append('<select class="form-control input-sm"  id="sel_team_id"></select>');
		
		for (var ele in teams) {
			var obj = teams[ele].split(":");
			$('#sel_team_id').append('<option value="' + obj[1] + '">' + obj[0] + '</option>');
		}
		// Filter results on select change
		$('#sel_team_id').on('change', function () {
			caption = $(this).find(":selected").text()
			value = $(this).find(":selected").val()
			if(!caption.includes("Choisis")){
				
				if(oTable != null){
					oTable.destroy();
				}else{
					$("#table-container").toggleClass("demo-html-visible");
				}
				
				//console.log( "ready!" );
				oTable = new DataTable('#example', {
					pageLength: 15,
					language: {
						url: '//cdn.datatables.net/plug-ins/1.13.6/i18n/fr-FR.json',
					},	
					order: [[0, "asc"]],
					ajax: {
						'url': '/upcoming/'+value,
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
					select: true,
					initComplete: function () {
					
					  this.api().rows().every( function ( rowIdx, tableLoop, rowLoop ) {
						var data = this.data();
						
						if (data.teamCaption === caption) {
						  this.select();
						}
					  });
					}
				});
			}
		});
	});
	  
});
