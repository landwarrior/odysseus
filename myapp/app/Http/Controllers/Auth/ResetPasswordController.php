<?php

namespace App\Http\Controllers\Auth;

use App\MstHr;
use App\Http\Controllers\Controller;
use App\Providers\RouteServiceProvider;
use Illuminate\Foundation\Auth\ResetsPasswords;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Hash;

class ResetPasswordController extends Controller
{
    /*
    |--------------------------------------------------------------------------
    | Password Reset Controller
    |--------------------------------------------------------------------------
    |
    | This controller is responsible for handling password reset requests
    | and uses a simple trait to include this behavior. You're free to
    | explore this trait and override any methods you wish to tweak.
    |
    */

    use ResetsPasswords;

    /**
     * Where to redirect users after resetting their password.
     *
     * @var string
     */
    protected $redirectTo = RouteServiceProvider::HOME;

    public function __construct()
    {
        $this->middleware('auth');
    }

    public function reset()
    {
        $user = Auth::user();
        if (empty($user)) {
            return redirect(route('login'));
        }

        return view('auth.passwords.reset', ['hr_cd' => $user->hr_cd]);
    }

    public function done(Request $request, $hr_cd)
    {
        $user = Auth::user();
        if (empty($user)) {
            return redirect(route('login'));
        }
        if ($user->hr_cd != $hr_cd) {
            // 不正行為をしたからお仕置き
            return redirect(route('logout'));
        }
        $validatedData = $request->validate([
            'password' => 'required|max:32|min:8',
            'password_confirmation' => 'required|max:32|min:8',
        ]);
        if ($request->password_confirmation != $request->password) {
            return redirect('/password')->with('not_match', '1');
        }

        DB::transaction(function () use ($request, $hr_cd) {
            $human = MstHr::findOrFail($hr_cd);
            $human->password = Hash::make($request->password);
            $human->save();
        });

        return redirect(route('home'));
    }
}
