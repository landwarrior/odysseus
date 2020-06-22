$('#conform-delete').on('input', function() {
  let inputText = $(this).val();
  let hrCd = $('#hr_cd').val();
  if (hrCd == inputText) {
    $('#delete-execute').attr('disabled', false);
  } else {
    $('#delete-execute').attr('disabled', true);
  }
});
