@extends('layouts.app')

@section('content')
<div class="container">
    <div class="row">
        <div class="col-md-12">
            <div class="card">
                <div class="card-header">{{ __('messages.project.list') }}</div>
                <div class="card-body">
                    <table class="table">
                        <tr class="text-center">
                            <th>{{ __('messages.project.no') }}</th>
                            <th>{{ __('messages.project.name') }}</th>
                            <th>{{ __('messages.project.order_amount') }}</th>
                            <th>{{ __('messages.project.from') }}</th>
                            <th>{{ __('messages.project.to') }}</th>
                        </tr>
                        @if(empty($projects['projects']))
                        <tr>
                            <td colspan="5">{{ __('messages.project.not_any') }}</td>
                        </tr>
                        @endif
                        @foreach($projects as $project)
                        <tr>
                            <td>{{ $project->project_no }}</td>
                            <td>{{ $project->name }}</td>
                            <td>{{ $project->order_amount }}</td>
                            <td>{{ $project->from_date }}</td>
                            <td>{{ $project->to_date }}</td>
                        </tr>
                        @endforeach
                    </table>
                    <div>
                        <a href="/project/create" class="btn btn-outline-primary">{{ __('messages.project.create') }}</a>
                    </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>
@endsection
