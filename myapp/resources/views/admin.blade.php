{{-- <div class="card-header">{{ __('messages.admin') }}</div> --}}
<div class="card-body">
  <!-- <div class="alert alert-danger" role="alert">
    進捗の芳しくないプロジェクトがあります！
  </div> -->
  <div class="container">
    <table class="table">
      <thead>
        <tr class="text-center">
          <th>プロジェクトNo</th>
          <th>プロジェクト名</th>
          <th>受注金額</th>
          <th>終了日</th>
          <th>予定工数</th>
          <th>実績工数</th>
        </tr>
      </thead>
      <tbody>
        @foreach($results as $result)
        <tr>
          <td>{{ $result['project_no'] }}</td>
          <td>{{ $result['name'] }}</td>
          <td class="text-right">￥ {{ number_format($result['order_amount']) }}</td>
          <td class="text-center">{{ $result['to_date'] }}</td>
          <td class="text-right">{{ $result['man_day'] }} 人日</td>
          <td class="text-right">{{ $result['result_day'] }} 人日</td>
        </tr>
        @endforeach
      </tbody>
    </table>
  </div>
</div>
