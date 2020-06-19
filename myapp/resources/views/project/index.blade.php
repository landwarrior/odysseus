@extends('layouts.app')

@section('content')
<div class="container">
  <div class="row">
    <div class="col-md-12">
      <div class="card">
        <div class="card-header">{{ __('messages.project.list') }}</div>
        <div class="card-body">
          <table class="table">
            <thead class="thead-light">
              <tr class="text-center">
                <th scope="col" style="width: 154px">{{ __('messages.project.no') }}</th>
                <th scope="col">{{ __('messages.project.name') }}</th>
                <th scope="col" style="width: 180px">{{ __('messages.project.order_amount') }}</th>
                <th scope="col" style="width: 154px">{{ __('messages.project.from') }}</th>
                <th scope="col" style="width: 154px">{{ __('messages.project.to') }}</th>
              </tr>
            </thead>
            <tbody>
              @if(count($projects) == 0)
              <tr class="text-center">
                <td colspan="5">{{ __('messages.project.not_any') }}</td>
              </tr>
              @endif
              @foreach($projects as $project)
              <tr class="text-center">
                <td>{{ $project->project_no }}</td>
                <td>{{ $project->name }}</td>
                <td>￥ {{ number_format($project->order_amount) }}</td>
                <td>{{ $project->from_date->format('Y/m/d') }}</td>
                <td>{{ $project->to_date->format('Y/m/d') }}</td>
              </tr>
              @endforeach
            </tbody>
          </table>
          <div class="row">
            <div class="col-md-6">
              <a href="/project/create" class="btn btn-outline-primary" role="button">{{ __('messages.create') }}</a>
            </div>
            <div class="col-md-6 text-right">
              <a href="/" class="btn btn-outline-secondary" role="button">{{ __('messages.back') }}</a>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
@endsection
