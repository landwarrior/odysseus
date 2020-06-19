$('.numFmt').on('focus', function() {
  $(this).val($(this).val().replace(/[^0-9.]/g, ''));
});
$('.numFmt').on('blur', function() {
  let num = Number.parseInt($(this).val());
  if (!isNaN(num)) {
    $(this).val( '￥ ' + num.toLocaleString());
  }
});
$(function() {
  // 初期表示時に金額がフォーマットされるようにする
  $('.numFmt').each(function() {
    let num = Number.parseInt($(this).val());
    if (!isNaN(num)) {
      $(this).val( '￥ ' + num.toLocaleString());
    }
  });
  // テキストボックスにカーソルを合わせると先頭にキャレットが置かれてしまうので直す
  $('input[type=text]').each(function() {
    let textElement = $(this).get(0);
    let len = textElement.value.length;
    textElement.setSelectionRange(len, len);
  });
  // テーブルの tr でリンクをつける
  $('table.table-hover tbody tr[data-href]').click(function() {
    window.location = $(this).attr('data-href');
  });
  // フラッシュメッセージはフェードアウトしてDOMも消す
  $('.flash-msg').fadeOut(3000, function() {
    $(this).remove();
  });
});
